// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package parser_test

import (
	"testing"

	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/parser/test_driver"
)

func mustParseOneStmt(t *testing.T, sql string) ast.StmtNode {
	t.Helper()
	stmt, err := parser.New().ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatalf("parse failed: %v\nsql: %s", err, sql)
	}
	return stmt
}

// 语法文档1（TXSQL/TDSQL-C）: CREATE 二级分区，SUBPARTITION BY RANGE/LIST + SUBPARTITION TEMPLATE
func TestTDSQLCreateSubpartitionTemplate(t *testing.T) {
	cases := []string{
		// 文档1原始示例: 一级RANGE + 二级RANGE模板 + ENGINE子句（含版本注释包装）
		"CREATE TABLE `t1` (" +
			"`id` int DEFAULT NULL, `purchased` int DEFAULT NULL, KEY `idx` (`id`,`purchased`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 " +
			"PARTITION BY RANGE (`id`) " +
			"SUBPARTITION BY RANGE (`purchased`) " +
			"SUBPARTITION TEMPLATE (" +
			"SUBPARTITION s0 VALUES LESS THAN (10) ENGINE = InnoDB," +
			"SUBPARTITION s1 VALUES LESS THAN (20) ENGINE = InnoDB) " +
			"(PARTITION p0 VALUES LESS THAN (10) ENGINE = InnoDB," +
			"PARTITION p1 VALUES LESS THAN (20) ENGINE = InnoDB," +
			"PARTITION p2 VALUES LESS THAN (30) ENGINE = InnoDB)",

		// 二级 LIST 模板 + VALUES IN
		"CREATE TABLE t2 (id int, city varchar(20), primary key(id, city)) " +
			"PARTITION BY RANGE (id) " +
			"SUBPARTITION BY LIST (length(city)) " +
			"SUBPARTITION TEMPLATE (" +
			"SUBPARTITION sa VALUES IN (1,2,3)," +
			"SUBPARTITION sb VALUES IN (4,5,6)) " +
			"(PARTITION p0 VALUES LESS THAN (100), PARTITION p1 VALUES LESS THAN (200))",

		// 模板子分区带 COMMENT / MAX_ROWS 等选项
		"CREATE TABLE t3 (id int, dt date, primary key(id, dt)) " +
			"PARTITION BY LIST (id) " +
			"SUBPARTITION BY RANGE (tdsql_month(dt)) " +
			"SUBPARTITION TEMPLATE (" +
			"SUBPARTITION s0 VALUES LESS THAN (202601) COMMENT = 'first' MAX_ROWS = 1000," +
			"SUBPARTITION s1 VALUES LESS THAN (202602)) " +
			"(PARTITION p0 VALUES IN (1), PARTITION p1 VALUES IN (2))",

		// MAXVALUE 上界
		"CREATE TABLE t4 (id int, dt date, primary key(id, dt)) " +
			"PARTITION BY LIST (id) " +
			"SUBPARTITION BY RANGE (tdsql_day(dt)) " +
			"SUBPARTITION TEMPLATE (" +
			"SUBPARTITION s0 VALUES LESS THAN (20260101)," +
			"SUBPARTITION smax VALUES LESS THAN MAXVALUE) " +
			"(PARTITION p0 VALUES IN (1))",
	}
	for i, sql := range cases {
		stmt := mustParseOneStmt(t, sql)
		ct, ok := stmt.(*ast.CreateTableStmt)
		if !ok {
			t.Fatalf("case %d: not CreateTableStmt", i)
		}
		if ct.Partition == nil || ct.Partition.Sub == nil {
			t.Fatalf("case %d: missing partition/subpartition info", i)
		}
		if len(ct.Partition.Sub.Template) == 0 {
			t.Fatalf("case %d: subpartition template not captured", i)
		}
	}
}

// 现场同构 DDL（已脱敏，表名/列名/注释均为虚构，结构与原始 DDL 一致）:
// 一级 LIST(murmurHashCodeAndMod) + 二级 RANGE(TDSQL_MONTH) + TEMPLATE + 32 个一级分区
func TestTDSQLCustomerSecondaryPartitionDDL(t *testing.T) {
	sql := "/*1:1000001-1000000001-1*/\n" + `create table  if not exists T_BIZ_DATA_HIST (
  id bigint(19) AUTO_INCREMENT NOT NULL COMMENT "xxx",
  src_region varchar (6) not null comment 'xxx',
  region varchar (6) not null comment 'xxx',
  item_code varchar (10) not null comment 'xxx',
  col_a varchar (3) not null comment 'xxx',
  col_b varchar (3) not null comment 'xxx',
  col_c varchar (2) not null comment 'xxx',
  col_d varchar (2) not null comment 'xxx',
  col_e varchar (32) not null comment 'xxx',
  col_f varchar (32) not null comment 'xxx',
  col_g varchar (100) not null comment 'xxx',
  col_h varchar (32) not null comment 'xxx',
  col_i varchar (32) not null comment 'xxx',
  biz_date date not null comment 'xxx',
  col_j varchar (32) not null comment 'xxx',
  col_k varchar (32) not null comment 'xxx',
  amount decimal (38, 2) not null comment 'xxx',
  col_l varchar (32) comment 'xxx',
  col_m varchar (32) comment 'xxx',
  gmt_create datetime(6) DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'xxx',
  gmt_modified datetime(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT 'xxx',
  primary key(
    id,
    region,
    biz_date
  )
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 collate = utf8mb4_bin comment 'xxx'  PARTITION BY LIST (murmurHashCodeAndMod(` + "`region`" + `,128))  SUBPARTITION BY RANGE(TDSQL_MONTH(biz_date))  SUBPARTITION TEMPLATE (
  SUBPARTITION p_auto_202601
  VALUES
    LESS THAN (202602),
    SUBPARTITION p_auto_202602
  VALUES
    LESS THAN (202603),
    SUBPARTITION p_auto_202603
  VALUES
    LESS THAN (202604)
)  (PARTITION p64 VALUES IN (64),PARTITION p65 VALUES IN (65),PARTITION p66 VALUES IN (66),PARTITION p67 VALUES IN (67),PARTITION p68 VALUES IN (68),PARTITION p69 VALUES IN (69),PARTITION p70 VALUES IN (70),PARTITION p71 VALUES IN (71),PARTITION p72 VALUES IN (72),PARTITION p73 VALUES IN (73),PARTITION p74 VALUES IN (74),PARTITION p75 VALUES IN (75),PARTITION p76 VALUES IN (76),PARTITION p77 VALUES IN (77),PARTITION p78 VALUES IN (78),PARTITION p79 VALUES IN (79),PARTITION p80 VALUES IN (80),PARTITION p81 VALUES IN (81),PARTITION p82 VALUES IN (82),PARTITION p83 VALUES IN (83),PARTITION p84 VALUES IN (84),PARTITION p85 VALUES IN (85),PARTITION p86 VALUES IN (86),PARTITION p87 VALUES IN (87),PARTITION p88 VALUES IN (88),PARTITION p89 VALUES IN (89),PARTITION p90 VALUES IN (90),PARTITION p91 VALUES IN (91),PARTITION p92 VALUES IN (92),PARTITION p93 VALUES IN (93),PARTITION p94 VALUES IN (94),PARTITION p95 VALUES IN (95))`

	stmt := mustParseOneStmt(t, sql)
	ct := stmt.(*ast.CreateTableStmt)
	if ct.Partition == nil || ct.Partition.Sub == nil {
		t.Fatal("missing partition/subpartition info")
	}
	if got := len(ct.Partition.Sub.Template); got != 3 {
		t.Fatalf("expect 3 template subpartitions, got %d", got)
	}
	if got := len(ct.Partition.Definitions); got != 32 {
		t.Fatalf("expect 32 partitions, got %d", got)
	}
	if ct.Partition.Sub.Template[0].Name.O != "p_auto_202601" {
		t.Fatalf("unexpected first template name: %s", ct.Partition.Sub.Template[0].Name.O)
	}
	if ct.Partition.Definitions[0].Name.O != "p64" || ct.Partition.Definitions[31].Name.O != "p95" {
		t.Fatal("unexpected partition names")
	}
}

// 语法文档1: ALTER TABLE 二级分区模板操作
func TestTDSQLAlterSubpartitionTemplate(t *testing.T) {
	cases := []struct {
		sql string
		tp  ast.AlterTableType
	}{
		{"ALTER TABLE t1 MODIFY PARTITION p0 TRUNCATE SUBPARTITION TEMPLATE s1", ast.AlterTableModifyPartitionTruncateSubpartitionTemplate},
		{"ALTER TABLE t1 TRUNCATE SUBPARTITION TEMPLATE s1", ast.AlterTableTruncateSubpartitionTemplate},
		{"ALTER TABLE t1 ADD SUBPARTITION TEMPLATE (SUBPARTITION s2 VALUES IN (1,2,3,4))", ast.AlterTableAddSubpartitionTemplate},
		{"ALTER TABLE t1 DROP SUBPARTITION TEMPLATE s2", ast.AlterTableDropSubpartitionTemplate},
	}
	for i, ca := range cases {
		stmt := mustParseOneStmt(t, ca.sql)
		at, ok := stmt.(*ast.AlterTableStmt)
		if !ok {
			t.Fatalf("case %d: not AlterTableStmt", i)
		}
		if len(at.Specs) != 1 || at.Specs[0].Tp != ca.tp {
			t.Fatalf("case %d: unexpected alter spec: %+v", i, at.Specs[0])
		}
	}
}

// 语法文档2 + 存量语法回归: shardkey / TDSQL_DISTRIBUTED / month() 等原有写法不允许回退
func TestTDSQLLegacySyntaxRegression(t *testing.T) {
	cases := []string{
		// 文档2: 一级Hash(shardkey) + LIST 分区
		"CREATE TABLE customers_1 (first_name VARCHAR(25) key, last_name VARCHAR(25), city VARCHAR(15), renewal DATE) " +
			"shardkey=first_name PARTITION BY LIST (city) (" +
			"PARTITION pRegion_1 VALUES IN('Beijing', 'Tianjin', 'Shanghai')," +
			"PARTITION pRegion_2 VALUES IN('Chongqing', 'Wulumuqi', 'Dalian'))",
		// 文档2: 一级Range，二级List（TDSQL_DISTRIBUTED 带定义列表）
		"CREATE TABLE tb_sub_r_l (id int(11) NOT NULL, order_id bigint NOT NULL, PRIMARY KEY (id,order_id)) " +
			"PARTITION BY list(order_id) (PARTITION p0 VALUES in (2121122), PARTITION p1 VALUES in (38937383)) " +
			"TDSQL_DISTRIBUTED BY RANGE(id) (s1 values less than (100),s2 values less than (1000))",
		// 文档2: month 函数分区
		"CREATE TABLE employees_int (id INT key NOT NULL, fname VARCHAR(30), hired date, store_id INT) " +
			"shardkey=id PARTITION BY RANGE ( month(hired) ) (" +
			"PARTITION p0 VALUES LESS THAN (199102), PARTITION p1 VALUES LESS THAN (199603), PARTITION p2 VALUES LESS THAN (200101))",
		// 原生 HASH 二级分区（MySQL 标准语法）不允许回退
		"CREATE TABLE ts (id INT, purchased DATE) PARTITION BY RANGE( YEAR(purchased) ) " +
			"SUBPARTITION BY HASH( TO_DAYS(purchased) ) SUBPARTITIONS 2 (" +
			"PARTITION p0 VALUES LESS THAN (1990), PARTITION p1 VALUES LESS THAN (2000))",
		// 每个一级分区内显式子分区定义（原有 SubPartDefinition 语法）
		"CREATE TABLE ts2 (id INT, purchased DATE) PARTITION BY RANGE( YEAR(purchased) ) " +
			"SUBPARTITION BY HASH( TO_DAYS(purchased) ) (" +
			"PARTITION p0 VALUES LESS THAN (1990) (SUBPARTITION s0, SUBPARTITION s1)," +
			"PARTITION p1 VALUES LESS THAN (2000) (SUBPARTITION s2, SUBPARTITION s3))",
	}
	for i, sql := range cases {
		stmt := mustParseOneStmt(t, sql)
		if _, ok := stmt.(*ast.CreateTableStmt); !ok {
			t.Fatalf("case %d: not CreateTableStmt", i)
		}
	}
}

// TEMPLATE 仍可作为普通标识符使用（非保留字回归）
func TestTemplateAsIdentifierRegression(t *testing.T) {
	cases := []string{
		"CREATE TABLE template (id int)",
		"SELECT template FROM t WHERE template = 1",
		"CREATE TABLE t (template int)",
	}
	for i, sql := range cases {
		mustParseOneStmt(t, sql)
		_ = i
	}
}

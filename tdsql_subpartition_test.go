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
	"strings"
	"testing"

	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	"github.com/pingcap/parser/model"
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

func restoreSQL(t *testing.T, stmt ast.StmtNode) string {
	t.Helper()
	var b strings.Builder
	if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &b)); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	return b.String()
}

// 文档3 标准例子的 round-trip：parse -> restore -> parse 必须等价。
// 当前期望失败的根因：CreateTableStmt.Restore 未输出 TDSQL_DISTRIBUTED 子句，
// 导致 restore 后再 parse 时遇到 TDSQL_PARTITION 没有 TDSQL_DISTRIBUTED 而报错。
func TestTDSQLDoc3RestoreRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{
			name: "doc3-employees: HASH dist + TDSQL_PARTITION BY RANGE",
			sql: "CREATE TABLE `employees` (`id` INT NOT NULL,`hired` DATE NOT NULL,`store_id` INT,PRIMARY KEY(`id`, `hired`)) " +
				"ENGINE = InnoDB " +
				"TDSQL_DISTRIBUTED BY HASH(`id`) " +
				"TDSQL_PARTITION BY RANGE (TDSQL_MONTH(`hired`)) " +
				"(PARTITION `s0` VALUES LESS THAN (199001), PARTITION `s1` VALUES LESS THAN (200001))",
		},
		{
			name: "doc3-customers_2: HASH dist + TDSQL_PARTITION BY LIST",
			sql: "CREATE TABLE `customers_2` (`id` INT,`city_id` INT,PRIMARY KEY(`id`, `city_id`)) " +
				"TDSQL_DISTRIBUTED BY HASH(`id`) " +
				"TDSQL_PARTITION BY LIST (`city_id`) " +
				"(PARTITION `sp1` VALUES IN (1, 2, 3), PARTITION `sp2` VALUES IN (4, 5, 6))",
		},
		{
			name: "legacy: TDSQL_DISTRIBUTED BY RANGE without TDSQL_PARTITION (回归)",
			sql: "CREATE TABLE `tb_sub_r_l` (`id` INT NOT NULL,`order_id` BIGINT NOT NULL,PRIMARY KEY(`id`, `order_id`)) " +
				"PARTITION BY LIST (`order_id`) (PARTITION `p0` VALUES IN (2121122),PARTITION `p1` VALUES IN (38937383)) " +
				"TDSQL_DISTRIBUTED BY RANGE(`id`) (`s1` VALUES LESS THAN (100),`s2` VALUES LESS THAN (1000))",
		},
	}

	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			st1 := mustParseOneStmt(t, ca.sql)
			ct1, ok := st1.(*ast.CreateTableStmt)
			if !ok {
				t.Fatalf("not CreateTableStmt")
			}
			if ct1.TdSqlDistributed == nil {
				t.Fatalf("expected TdSqlDistributed != nil after first parse")
			}

			restored := restoreSQL(t, st1)
			if !strings.Contains(strings.ToUpper(restored), "TDSQL_DISTRIBUTED") {
				t.Fatalf("restored SQL missing TDSQL_DISTRIBUTED clause:\n%s", restored)
			}

			st2, err := parser.New().ParseOneStmt(restored, "", "")
			if err != nil {
				t.Fatalf("re-parse restored SQL failed: %v\nrestored: %s", err, restored)
			}
			ct2, ok := st2.(*ast.CreateTableStmt)
			if !ok {
				t.Fatalf("re-parsed stmt is not CreateTableStmt")
			}
			if ct2.TdSqlDistributed == nil {
				t.Fatalf("re-parsed AST lost TdSqlDistributed")
			}
			if ct1.TdSqlDistributed.Tp != ct2.TdSqlDistributed.Tp {
				t.Fatalf("dist Tp mismatch: %v vs %v", ct1.TdSqlDistributed.Tp, ct2.TdSqlDistributed.Tp)
			}

			if ct1.Partition != nil && ct1.Partition.Sub != nil {
				if ct2.Partition == nil || ct2.Partition.Sub == nil {
					t.Fatalf("re-parsed AST lost TDSQL_PARTITION sub clause")
				}
				if ct1.Partition.Sub.Tp != ct2.Partition.Sub.Tp {
					t.Fatalf("sub Tp mismatch: %v vs %v", ct1.Partition.Sub.Tp, ct2.Partition.Sub.Tp)
				}
				if len(ct1.Partition.Sub.Template) != len(ct2.Partition.Sub.Template) {
					t.Fatalf("sub template length mismatch: %d vs %d",
						len(ct1.Partition.Sub.Template), len(ct2.Partition.Sub.Template))
				}
			}
		})
	}
}

// 仅 TdSqlDistributed.Restore 单元行为（HASH 分支必须输出 HASH(expr)）。
func TestTdSqlDistributedRestoreHash(t *testing.T) {
	sql := "CREATE TABLE t (id int) TDSQL_DISTRIBUTED BY HASH(id)"
	st := mustParseOneStmt(t, sql)
	ct := st.(*ast.CreateTableStmt)
	if ct.TdSqlDistributed == nil || ct.TdSqlDistributed.Tp != model.PartitionTypeHash {
		t.Fatalf("unexpected dist: %+v", ct.TdSqlDistributed)
	}
	out := restoreSQL(t, st)
	upper := strings.ToUpper(out)
	if !strings.Contains(upper, "TDSQL_DISTRIBUTED BY HASH") {
		t.Fatalf("restored SQL missing 'TDSQL_DISTRIBUTED BY HASH':\n%s", out)
	}
	if _, err := parser.New().ParseOneStmt(out, "", ""); err != nil {
		t.Fatalf("re-parse failed: %v\n%s", err, out)
	}
}

func findIndexConstraint(t *testing.T, stmt *ast.CreateTableStmt, name string) *ast.Constraint {
	t.Helper()
	for _, c := range stmt.Constraints {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("index constraint %q not found", name)
	return nil
}

func TestTDSQLDoc3SetGlobalIndex(t *testing.T) {
	cases := []struct {
		sql       string
		setGlobal map[string]bool
	}{
		{
			sql: "CREATE TABLE customers_1 (" +
				"id int, store_id int, " +
				"primary key(id), " +
				"index idx1(store_id) SET_GLOBAL" +
				") ENGINE=InnoDB tdsql_distributed by hash(id)",
			setGlobal: map[string]bool{"idx1": true},
		},
		{
			sql: "CREATE TABLE employees (" +
				"id INT NOT NULL, fname VARCHAR(30), hired DATE NOT NULL DEFAULT '9999-12-31', " +
				"separated DATE NOT NULL DEFAULT '9999-12-31', store_id INT, " +
				"Primary key(id, hired), INDEX idx(store_id) SET_GLOBAL" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 " +
				"TDSQL_DISTRIBUTED BY HASH (id) " +
				"TDSQL_PARTITION BY RANGE ( tdsql_month(hired) ) (" +
				"PARTITION s0 VALUES LESS THAN (199001), " +
				"PARTITION s1 VALUES LESS THAN (200001), " +
				"PARTITION s2 VALUES LESS THAN (201001))",
			setGlobal: map[string]bool{"idx": true},
		},
		{
			sql: "CREATE TABLE customers_2 (" +
				"id int, city_id int, store_id int, bag_id int, birthday DATE, " +
				"primary key(id, city_id), " +
				"index idx1(store_id) SET_GLOBAL, " +
				"index bag_idx2(bag_id) SET_GLOBAL, " +
				"index bri_idx(birthday, id, city_id)" +
				") tdsql_distributed by hash(id) " +
				"TDSQL_PARTITION BY LIST (city_id) (" +
				"PARTITION spRegion_1 VALUES IN(1, 2, 3), " +
				"PARTITION spRegion_2 VALUES IN(4, 5, 6), " +
				"PARTITION spRegion_3 VALUES IN(7, 8, 9), " +
				"PARTITION spRegion_4 VALUES IN(10, 11, 12))",
			setGlobal: map[string]bool{
				"idx1":     true,
				"bag_idx2": true,
				"bri_idx":  false,
			},
		},
	}
	for i, ca := range cases {
		stmt := mustParseOneStmt(t, ca.sql).(*ast.CreateTableStmt)
		for name, want := range ca.setGlobal {
			c := findIndexConstraint(t, stmt, name)
			if c.Option == nil {
				if want {
					t.Fatalf("case %d: index %q: Option is nil, want SetGlobal=%v", i, name, want)
				}
				continue
			}
			if c.Option.SetGlobal != want {
				t.Fatalf("case %d: index %q: SetGlobal=%v, want %v", i, name, c.Option.SetGlobal, want)
			}
		}
	}
}

func TestTDSQLDoc3SetGlobalIdentifierRegression(t *testing.T) {
	cases := []string{
		"CREATE TABLE set_global (id int)",
		"CREATE TABLE tdsql_partition (id int)",
		"CREATE TABLE customers_1 (" +
			"id int, store_id int, " +
			"primary key(id), " +
			"index idx1(store_id)" +
			") ENGINE=InnoDB tdsql_distributed by hash(id)",
	}
	for i, sql := range cases {
		mustParseOneStmt(t, sql)
		_ = i
	}
}

func TestTDSQLDoc3DistributedByHash(t *testing.T) {
	cases := []string{
		"CREATE TABLE customers_1 (" +
			"id int, store_id int, primary key(id), index idx1(store_id) SET_GLOBAL" +
			") ENGINE=InnoDB tdsql_distributed by hash(id)",
		"CREATE TABLE employees (" +
			"id INT NOT NULL, hired DATE NOT NULL DEFAULT '9999-12-31', store_id INT, " +
			"Primary key(id, hired), INDEX idx(store_id) SET_GLOBAL" +
			") ENGINE=InnoDB TDSQL_DISTRIBUTED BY HASH (id) " +
			"TDSQL_PARTITION BY RANGE ( tdsql_month(hired) ) (" +
			"PARTITION s0 VALUES LESS THAN (199001))",
		"CREATE TABLE customers_2 (" +
			"id int, city_id int, store_id int, " +
			"primary key(id, city_id), index idx1(store_id) SET_GLOBAL" +
			") tdsql_distributed by hash(id) " +
			"TDSQL_PARTITION BY LIST (city_id) (" +
			"PARTITION spRegion_1 VALUES IN(1, 2, 3))",
	}
	for i, sql := range cases {
		stmt := mustParseOneStmt(t, sql).(*ast.CreateTableStmt)
		if stmt.TdSqlDistributed == nil {
			t.Fatalf("case %d: TdSqlDistributed is nil", i)
		}
		method := stmt.TdSqlDistributed.PartitionMethod
		if method.Tp != model.PartitionTypeHash {
			t.Fatalf("case %d: partition type=%v, want HASH", i, method.Tp)
		}
		if method.Expr == nil {
			t.Fatalf("case %d: HASH expr is nil", i)
		}
	}
}

func TestTDSQLDoc3PartitionBySubpartition(t *testing.T) {
	t.Run("RANGE", func(t *testing.T) {
		sql := "CREATE TABLE employees (" +
			"id INT NOT NULL, fname VARCHAR(30), hired DATE NOT NULL DEFAULT '9999-12-31', " +
			"separated DATE NOT NULL DEFAULT '9999-12-31', store_id INT, " +
			"Primary key(id, hired), INDEX idx(store_id) SET_GLOBAL" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 " +
			"TDSQL_DISTRIBUTED BY HASH (id) " +
			"TDSQL_PARTITION BY RANGE ( tdsql_month(hired) ) (" +
			"PARTITION s0 VALUES LESS THAN (199001), " +
			"PARTITION s1 VALUES LESS THAN (200001), " +
			"PARTITION s2 VALUES LESS THAN (201001))"
		stmt := mustParseOneStmt(t, sql).(*ast.CreateTableStmt)
		if stmt.Partition == nil || stmt.Partition.Sub == nil {
			t.Fatal("missing Partition.Sub")
		}
		sub := stmt.Partition.Sub
		if sub.Tp != model.PartitionTypeRange {
			t.Fatalf("sub partition type=%v, want RANGE", sub.Tp)
		}
		if sub.Expr == nil {
			t.Fatal("sub partition expr is nil")
		}
		if len(sub.Template) != 3 {
			t.Fatalf("template count=%d, want 3", len(sub.Template))
		}
		if sub.Template[0].Name.O != "s0" {
			t.Fatalf("first template name=%q, want s0", sub.Template[0].Name.O)
		}
		if _, ok := sub.Template[0].Clause.(*ast.PartitionDefinitionClauseLessThan); !ok {
			t.Fatalf("first template clause type=%T, want *PartitionDefinitionClauseLessThan", sub.Template[0].Clause)
		}
	})

	t.Run("LIST", func(t *testing.T) {
		sql := "CREATE TABLE customers_2 (" +
			"id int, city_id int, store_id int, bag_id int, birthday DATE, " +
			"primary key(id, city_id), " +
			"index idx1(store_id) SET_GLOBAL, " +
			"index bag_idx2(bag_id) SET_GLOBAL, " +
			"index bri_idx(birthday, id, city_id)" +
			") tdsql_distributed by hash(id) " +
			"TDSQL_PARTITION BY LIST (city_id) (" +
			"PARTITION spRegion_1 VALUES IN(1, 2, 3), " +
			"PARTITION spRegion_2 VALUES IN(4, 5, 6), " +
			"PARTITION spRegion_3 VALUES IN(7, 8, 9), " +
			"PARTITION spRegion_4 VALUES IN(10, 11, 12))"
		stmt := mustParseOneStmt(t, sql).(*ast.CreateTableStmt)
		if stmt.Partition == nil || stmt.Partition.Sub == nil {
			t.Fatal("missing Partition.Sub")
		}
		sub := stmt.Partition.Sub
		if sub.Tp != model.PartitionTypeList {
			t.Fatalf("sub partition type=%v, want LIST", sub.Tp)
		}
		if len(sub.Template) != 4 {
			t.Fatalf("template count=%d, want 4", len(sub.Template))
		}
		if sub.Template[0].Name.O != "spRegion_1" {
			t.Fatalf("first template name=%q, want spRegion_1", sub.Template[0].Name.O)
		}
		if _, ok := sub.Template[0].Clause.(*ast.PartitionDefinitionClauseIn); !ok {
			t.Fatalf("first template clause type=%T, want *PartitionDefinitionClauseIn", sub.Template[0].Clause)
		}
	})
}

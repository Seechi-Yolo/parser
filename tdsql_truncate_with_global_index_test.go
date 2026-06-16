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
	"bytes"
	"strings"
	"testing"

	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	_ "github.com/pingcap/parser/test_driver"
)

func truncatePartitionSpec(t *testing.T, sql string) *ast.AlterTableSpec {
	t.Helper()
	stmt := mustParseOneStmt(t, sql)
	at, ok := stmt.(*ast.AlterTableStmt)
	if !ok {
		t.Fatalf("not AlterTableStmt: %T", stmt)
	}
	if len(at.Specs) != 1 {
		t.Fatalf("expected 1 alter spec, got %d", len(at.Specs))
	}
	spec := at.Specs[0]
	if spec.Tp != ast.AlterTableTruncatePartition {
		t.Fatalf("unexpected alter spec type: %v", spec.Tp)
	}
	return spec
}

func TestTruncateWithGlobalIndex(t *testing.T) {
	cases := []struct {
		sql              string
		withGlobalIndex  bool
		onAllPartitions  bool
		partitionNames   []string
	}{
		{
			sql:             "ALTER TABLE t1 TRUNCATE PARTITION p0 WITH GLOBAL INDEX",
			withGlobalIndex: true,
			partitionNames:  []string{"p0"},
		},
		{
			sql:             "ALTER TABLE t1 TRUNCATE PARTITION p0, p1 WITH GLOBAL INDEX",
			withGlobalIndex: true,
			partitionNames:  []string{"p0", "p1"},
		},
		{
			sql:             "ALTER TABLE t1 TRUNCATE PARTITION ALL WITH GLOBAL INDEX",
			withGlobalIndex: true,
			onAllPartitions: true,
		},
	}
	for i, ca := range cases {
		t.Run(ca.sql, func(t *testing.T) {
			spec := truncatePartitionSpec(t, ca.sql)
			if spec.WithGlobalIndex != ca.withGlobalIndex {
				t.Fatalf("case %d: WithGlobalIndex=%v, want %v", i, spec.WithGlobalIndex, ca.withGlobalIndex)
			}
			if spec.OnAllPartitions != ca.onAllPartitions {
				t.Fatalf("case %d: OnAllPartitions=%v, want %v", i, spec.OnAllPartitions, ca.onAllPartitions)
			}
			if ca.onAllPartitions {
				return
			}
			if len(spec.PartitionNames) != len(ca.partitionNames) {
				t.Fatalf("case %d: partition count=%d, want %d", i, len(spec.PartitionNames), len(ca.partitionNames))
			}
			for j, name := range ca.partitionNames {
				if spec.PartitionNames[j].O != name {
					t.Fatalf("case %d: partition[%d]=%q, want %q", i, j, spec.PartitionNames[j].O, name)
				}
			}
		})
	}
}

func TestTruncateWithGlobalIndexRegression(t *testing.T) {
	cases := []struct {
		sql            string
		partitionNames []string
	}{
		{
			sql:            "ALTER TABLE t1 TRUNCATE PARTITION p0",
			partitionNames: []string{"p0"},
		},
		{
			sql:            "ALTER TABLE t1 TRUNCATE PARTITION p0, p1",
			partitionNames: []string{"p0", "p1"},
		},
		{
			sql: "ALTER TABLE t1 TRUNCATE PARTITION ALL",
		},
	}
	for i, ca := range cases {
		spec := truncatePartitionSpec(t, ca.sql)
		if spec.WithGlobalIndex {
			t.Fatalf("case %d: WithGlobalIndex=true, want false", i)
		}
		if ca.partitionNames == nil {
			if !spec.OnAllPartitions {
				t.Fatalf("case %d: OnAllPartitions=false, want true", i)
			}
			continue
		}
		if len(spec.PartitionNames) != len(ca.partitionNames) {
			t.Fatalf("case %d: partition count=%d, want %d", i, len(spec.PartitionNames), len(ca.partitionNames))
		}
	}
}

func TestTruncateWithGlobalIndexRestoreRoundTrip(t *testing.T) {
	sql := "ALTER TABLE t1 TRUNCATE PARTITION p0 WITH GLOBAL INDEX"
	stmt := mustParseOneStmt(t, sql).(*ast.AlterTableStmt)

	var buf bytes.Buffer
	ctx := format.NewRestoreCtx(format.RestoreKeyWordUppercase|format.RestoreNameBackQuotes, &buf)
	if err := stmt.Restore(ctx); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	restored := buf.String()
	if !strings.Contains(restored, "WITH GLOBAL INDEX") {
		t.Fatalf("restored SQL %q does not contain WITH GLOBAL INDEX", restored)
	}

	stmt2 := mustParseOneStmt(t, restored).(*ast.AlterTableStmt)
	if len(stmt2.Specs) != 1 {
		t.Fatalf("expected 1 spec after round-trip, got %d", len(stmt2.Specs))
	}
	spec := stmt2.Specs[0]
	if !spec.WithGlobalIndex {
		t.Fatal("WithGlobalIndex=false after round-trip, want true")
	}
	if spec.Tp != ast.AlterTableTruncatePartition {
		t.Fatalf("unexpected spec type after round-trip: %v", spec.Tp)
	}
	if len(spec.PartitionNames) != 1 || spec.PartitionNames[0].O != "p0" {
		t.Fatalf("unexpected partition names after round-trip: %+v", spec.PartitionNames)
	}
}

func TestTruncateWithGlobalIndexIdentifierRegression(t *testing.T) {
	cases := []string{
		"CREATE TABLE global (id int)",
		"CREATE TABLE t (id int, INDEX idx(id))",
	}
	for _, sql := range cases {
		mustParseOneStmt(t, sql)
	}
}

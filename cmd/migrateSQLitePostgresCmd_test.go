package cmd

import (
	"bytes"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/dbcopy"
	"strings"
	"testing"
)

func TestSQLitePostgresAppendFlagAndValidation(t *testing.T) {
	t.Setenv("AQUALOG_DATABASE_URL", "")
	t.Setenv("AQUALOG_DB", "")
	db.SetDatabaseURLOverride("")
	db.SetPathOverride("")
	t.Cleanup(func() { db.SetDatabaseURLOverride(""); db.SetPathOverride("") })
	command := newSQLiteToPostgresCommand()
	if value, err := command.Flags().GetBool("append"); err != nil || value {
		t.Fatalf("append default = %v, %v", value, err)
	}
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--source", "unused.db", "--append"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "requires --database-url") {
		t.Fatalf("backend error = %v", err)
	}
	if strings.Contains(output.String(), "successfully") {
		t.Fatal("success printed on failure")
	}
	command = newSQLiteToPostgresCommand()
	command.SetArgs([]string{"--append"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("missing source error = %v", err)
	}
}

func TestWriteAppendResult(t *testing.T) {
	command := newSQLiteToPostgresCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	writeAppendResult(command, dbcopy.AppendResult{Counts: map[string]int64{"sites": 3, "projects": 2}, ProjectIDs: map[int64]int64{42: 99, 1: 98}})
	want := "SQLite database appended to PostgreSQL successfully\nprojects: 2 rows imported\nsites: 3 rows imported\nProject ID 1 -> 98\nProject ID 42 -> 99\n"
	if output.String() != want {
		t.Fatalf("output = %q", output.String())
	}
}

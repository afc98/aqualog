package dbcopy

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenSQLiteReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source database.sqlite")
	writable, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writable.Exec("CREATE TABLE sample (id INTEGER PRIMARY KEY)"); err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") {
			t.Skip("go-sqlite3 requires CGO")
		}
		t.Fatal(err)
	}
	if _, err := writable.Exec("INSERT INTO sample(id) VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	if err := writable.Close(); err != nil {
		t.Fatal(err)
	}

	readonly, err := OpenSQLiteReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer readonly.Close()
	var count int
	if err := readonly.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sample").Scan(&count); err != nil {
		t.Fatalf("read from source: %v", err)
	}
	if count != 1 {
		t.Fatalf("got %d rows, want 1", count)
	}
	if _, err := readonly.Exec("INSERT INTO sample(id) VALUES (2)"); err == nil {
		t.Fatal("write through read-only connection unexpectedly succeeded")
	}
}

func TestOpenSQLiteReadOnlyRejectsMissingSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.sqlite")
	database, err := OpenSQLiteReadOnly(path)
	if database != nil {
		database.Close()
	}
	if err == nil {
		t.Fatal("missing source unexpectedly opened")
	}
}

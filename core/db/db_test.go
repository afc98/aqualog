package db

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestResolvePathFlagWinsOverEnv(t *testing.T) {
	getenv := mapEnv(map[string]string{
		"AQUALOG_DB": "/env/aqualog.db",
	})

	info, err := resolvePath("relative/project.db", getenv, "linux", testHome("/home/test"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Source != PathSourceFlag {
		t.Fatalf("source = %q; want %q", info.Source, PathSourceFlag)
	}
	if info.Path != filepath.Clean("relative/project.db") {
		t.Fatalf("path = %q; want %q", info.Path, filepath.Clean("relative/project.db"))
	}
}

func TestResolvePathEnvWinsOverDefault(t *testing.T) {
	getenv := mapEnv(map[string]string{
		"AQUALOG_DB": "/env/aqualog.db",
	})

	info, err := resolvePath("", getenv, "linux", testHome("/home/test"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Source != PathSourceEnv {
		t.Fatalf("source = %q; want %q", info.Source, PathSourceEnv)
	}
	if info.Path != filepath.Clean("/env/aqualog.db") {
		t.Fatalf("path = %q; want %q", info.Path, filepath.Clean("/env/aqualog.db"))
	}
}

func TestResolvePathLinuxUsesXDGDataHome(t *testing.T) {
	getenv := mapEnv(map[string]string{
		"XDG_DATA_HOME": "/data/home",
	})

	info, err := resolvePath("", getenv, "linux", testHome("/home/test"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/data/home", "aqualog", "aqualog.db")
	if info.Source != PathSourceDefault {
		t.Fatalf("source = %q; want %q", info.Source, PathSourceDefault)
	}
	if info.Path != want {
		t.Fatalf("path = %q; want %q", info.Path, want)
	}
}

func TestResolvePathDefaultsByOS(t *testing.T) {
	tests := []struct {
		name string
		goos string
		env  map[string]string
		home string
		want string
	}{
		{
			name: "windows local app data",
			goos: "windows",
			env:  map[string]string{"LOCALAPPDATA": filepath.Join("C:", "Users", "test", "AppData", "Local")},
			home: filepath.Join("C:", "Users", "test"),
			want: filepath.Join("C:", "Users", "test", "AppData", "Local", "Aqualog", "aqualog.db"),
		},
		{
			name: "mac application support",
			goos: "darwin",
			home: "/Users/test",
			want: filepath.Join("/Users/test", "Library", "Application Support", "Aqualog", "aqualog.db"),
		},
		{
			name: "linux home data",
			goos: "linux",
			home: "/home/test",
			want: filepath.Join("/home/test", ".local", "share", "aqualog", "aqualog.db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := resolvePath("", mapEnv(tt.env), tt.goos, testHome(tt.home))
			if err != nil {
				t.Fatal(err)
			}
			if info.Source != PathSourceDefault {
				t.Fatalf("source = %q; want %q", info.Source, PathSourceDefault)
			}
			if info.Path != tt.want {
				t.Fatalf("path = %q; want %q", info.Path, tt.want)
			}
		})
	}
}

func TestResolvePathDefaultHomeError(t *testing.T) {
	_, err := resolvePath("", mapEnv(nil), "linux", func() (string, error) {
		return "", errors.New("home unavailable")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSchemaCreatesFileRemovalIndexes(t *testing.T) {
	SetPathOverride(filepath.Join(t.TempDir(), "aqualog.db"))
	t.Cleanup(func() { SetPathOverride("") })

	dbConn, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()

	for _, name := range []string{
		"idx_logger_data_logger_file_id",
		"idx_corrected_data_site_id",
		"idx_corrected_data_logger_data_id",
	} {
		if !indexExists(t, dbConn, name) {
			t.Errorf("index %q does not exist", name)
		}
	}
}

func indexExists(t *testing.T, dbConn *sql.DB, name string) bool {
	t.Helper()
	var exists bool
	if err := dbConn.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'index' AND name = ?)`,
		name,
	).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}

func mapEnv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func testHome(home string) func() (string, error) {
	return func() (string, error) {
		return home, nil
	}
}

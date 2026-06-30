package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func WriteTempFile(t testing.TB, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func WriteConfig(t testing.TB, content string) string {
	t.Helper()
	return WriteTempFile(t, t.TempDir(), "velocirepo.toml", content)
}

func AssertFileExists(t testing.TB, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %s", path)
	}
}

func AssertFileNotExists(t testing.TB, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file not to exist: %s", path)
	}
}

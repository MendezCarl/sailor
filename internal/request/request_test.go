package request

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "req.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFile_SchemaVersionParsed(t *testing.T) {
	path := writeFixture(t, "schema_version: 1\nmethod: GET\nurl: https://example.com\n")
	req, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", req.SchemaVersion)
	}
}

func TestLoadFile_SchemaVersionAbsent(t *testing.T) {
	path := writeFixture(t, "method: GET\nurl: https://example.com\n")
	req, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.SchemaVersion != 0 {
		t.Errorf("SchemaVersion = %d, want 0", req.SchemaVersion)
	}
}

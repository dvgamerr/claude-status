package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCreatesAndReplacesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested.json")
	for _, content := range []string{"first", "second"} {
		if err := Write(path, []byte(content), 0o600); err != nil {
			t.Fatalf("Write(%q) error = %v", content, err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != content {
			t.Fatalf("content = %q, want %q", got, content)
		}
	}
}

func TestWriteRejectsEmptyPathAndMissingDirectory(t *testing.T) {
	if err := Write("", nil, 0o600); err == nil {
		t.Fatal("expected empty path error")
	}
	if err := Write(filepath.Join(t.TempDir(), "missing", "file"), nil, 0o600); err == nil {
		t.Fatal("expected missing directory error")
	}
}

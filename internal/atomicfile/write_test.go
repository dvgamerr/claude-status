package atomicfile

import (
	"os"
	"path/filepath"
	"strings"
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

// TestWriteReportsRenameErrorAndCleansUpTempFile pins the replace-step error
// branch: renaming the finished temp file onto a path that is already an
// existing directory fails on both Unix (EISDIR) and Windows (Access is
// denied) without needing any platform-specific permission setup, and the
// deferred cleanup must still remove the now-orphaned temp file rather than
// leaking it into the destination directory.
func TestWriteReportsRenameErrorAndCleansUpTempFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	err := Write(target, []byte("payload"), 0o600)
	if err == nil || !strings.Contains(err.Error(), "replace") {
		t.Fatalf("Write() error = %v, want a replace error", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "target" {
			t.Fatalf("Write() leaked temp entry %q after a rename failure", entry.Name())
		}
	}
}

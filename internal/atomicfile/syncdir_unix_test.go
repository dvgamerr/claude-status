//go:build !windows

package atomicfile

import (
	"path/filepath"
	"testing"
)

func TestSyncDirMissingDirectory(t *testing.T) {
	if err := syncDir(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("syncDir() on a missing directory: error = nil")
	}
}

func TestSyncDirSuccess(t *testing.T) {
	if err := syncDir(t.TempDir()); err != nil {
		t.Fatalf("syncDir() error = %v", err)
	}
}

//go:build windows

package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// TestReadFileWithRetryWrapsWindowsSharingViolation verifies the
// isTransientReadErr(err) branch inside readFileWithRetry (store.go) that
// wraps a persistent Windows sharing/lock violation with ErrTransientRead.
// It opens the target file exclusively (share mode 0) via a raw
// windows.CreateFile handle and keeps it open for the whole retry window, so
// every readLimitedFile attempt inside readFileWithRetry hits
// ERROR_SHARING_VIOLATION instead of succeeding.
func TestReadFileWithRetryWrapsWindowsSharingViolation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "locked.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	namePtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(
		namePtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // exclusive: no FILE_SHARE_* flags, so a concurrent os.Open fails.
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatalf("windows.CreateFile() error = %v", err)
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	start := time.Now()
	_, err = readFileWithRetry(path)
	if err == nil {
		t.Fatal("readFileWithRetry() error = nil, want a sharing-violation error")
	}
	if !errors.Is(err, ErrTransientRead) {
		t.Fatalf("readFileWithRetry() error = %v, want it wrapped with ErrTransientRead", err)
	}
	minElapsed := readRetryDelay * (readRetryAttempts - 1)
	if elapsed := time.Since(start); elapsed < minElapsed {
		t.Fatalf("readFileWithRetry() returned after %v, want at least %v of retry sleeps", elapsed, minElapsed)
	}
}

//go:build windows

package state

import (
	"errors"
	"fmt"
	"testing"

	"golang.org/x/sys/windows"
)

func TestIsTransientReadErr(t *testing.T) {
	if !isTransientReadErr(fmt.Errorf("wrapped: %w", windows.ERROR_SHARING_VIOLATION)) {
		t.Fatal("sharing violation was not transient")
	}
	if !isTransientReadErr(windows.ERROR_LOCK_VIOLATION) {
		t.Fatal("lock violation was not transient")
	}
	if isTransientReadErr(errors.New("ordinary")) || isTransientReadErr(windows.ERROR_FILE_NOT_FOUND) {
		t.Fatal("ordinary error was transient")
	}
}

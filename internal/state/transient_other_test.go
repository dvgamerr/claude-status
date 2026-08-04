//go:build !windows

package state

import (
	"errors"
	"fmt"
	"testing"
)

// TestIsTransientReadErrAlwaysFalseOnNonWindows exercises the !windows
// implementation of isTransientReadErr (transient_other.go), whose only
// known transient cause (a Windows sharing/lock violation, see
// transient_windows.go) does not exist on this platform, so every error —
// including nil, a plain error, and a wrapped error — must report false.
//
// NOTE: this file only compiles into the test binary on non-Windows
// platforms. It needs to be run on a Linux (or other non-Windows) box to
// actually verify transient_other.go's coverage; a Windows-only local run
// cannot exercise it because transient_windows.go's isTransientReadErr is the
// one linked into a Windows build.
func TestIsTransientReadErrAlwaysFalseOnNonWindows(t *testing.T) {
	cases := []error{
		nil,
		errors.New("ordinary error"),
		fmt.Errorf("wrapped: %w", errors.New("inner")),
		errors.New(""),
	}
	for _, err := range cases {
		if isTransientReadErr(err) {
			t.Fatalf("isTransientReadErr(%v) = true, want false on non-Windows", err)
		}
	}
}

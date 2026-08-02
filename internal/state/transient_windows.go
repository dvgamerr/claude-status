//go:build windows

package state

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// isTransientReadErr reports whether err is the Windows sharing/lock
// violation a reader can hit landing mid-rename (see writeAtomic's
// os.Rename), which readRetryAttempts already retries but doesn't always
// clear before this call reports its final failure. These constants live in
// golang.org/x/sys/windows rather than the standard syscall package.
func isTransientReadErr(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	return errno == windows.ERROR_SHARING_VIOLATION || errno == windows.ERROR_LOCK_VIOLATION
}

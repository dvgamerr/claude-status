//go:build !windows

package state

// isTransientReadErr's only known cause (see the windows build of this file)
// is a Windows-specific sharing/lock violation, so non-Windows platforms
// never classify a read failure as transient.
func isTransientReadErr(_ error) bool {
	return false
}

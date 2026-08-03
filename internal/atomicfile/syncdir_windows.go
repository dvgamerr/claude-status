//go:build windows

package atomicfile

// Windows does not expose a portable directory fsync operation.
func syncDir(string) error { return nil }

//go:build !windows

package service

import "context"

// IsWindowsService is always false outside Windows: main() uses this to
// decide whether to enter the SCM dispatch loop before doing anything
// else, a concept that only exists on Windows. On Linux/macOS the service
// manager (systemd/launchd) runs this binary as a plain foreground process,
// no special protocol required.
func IsWindowsService() bool { return false }

// RunAsService is never actually called on this platform (IsWindowsService
// is always false), but must exist so main.go compiles here too.
func RunAsService(string, func(ctx context.Context) error) error {
	return ErrUnsupportedPlatform
}

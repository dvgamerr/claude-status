// Package service installs claude-status's relay as a background service
// using whatever the host OS's own service manager is: a real Windows
// Service on Windows, a systemd --user unit on Linux, and a launchd
// LaunchAgent on macOS. One Config and one set of Install/Remove/Start/
// Stop/Status calls cover all three; the platform-specific mechanics live
// in the manager_<os>.go files, matching this repo's existing per-OS
// pattern (see internal/touch).
package service

import "fmt"

// Config describes the background command the OS service manager should
// supervise. Args are passed to this same executable (os.Executable()) —
// e.g. {"relay", "--mirror-ssh", "pilab"} — so the installed service runs
// exactly the CLI command a user could run in a terminal themselves.
type Config struct {
	Name        string
	DisplayName string
	Description string
	Args        []string
}

// State is the coarse status of a previously installed service.
type State int

const (
	StateNotInstalled State = iota
	StateStopped
	StateRunning
)

func (s State) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StateStopped:
		return "stopped"
	default:
		return "not installed"
	}
}

// ErrUnsupportedPlatform is returned by manager_other.go on any OS this
// package doesn't yet implement a service manager for.
var ErrUnsupportedPlatform = fmt.Errorf("service management is not supported on this platform")

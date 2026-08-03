//go:build !windows && !linux && !darwin

package service

// Install reports that service management is unsupported.
func Install(Config) error { return ErrUnsupportedPlatform }

// Remove reports that service management is unsupported.
func Remove(string) error { return ErrUnsupportedPlatform }

// Start reports that service management is unsupported.
func Start(string) error { return ErrUnsupportedPlatform }

// Stop reports that service management is unsupported.
func Stop(string) error { return ErrUnsupportedPlatform }

// Status reports that service management is unsupported.
func Status(string) (State, error) { return StateNotInstalled, ErrUnsupportedPlatform }

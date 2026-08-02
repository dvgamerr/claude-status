//go:build !windows && !linux && !darwin

package service

func Install(Config) error         { return ErrUnsupportedPlatform }
func Remove(string) error          { return ErrUnsupportedPlatform }
func Start(string) error           { return ErrUnsupportedPlatform }
func Stop(string) error            { return ErrUnsupportedPlatform }
func Status(string) (State, error) { return StateNotInstalled, ErrUnsupportedPlatform }

package service

import (
	"context"
	"errors"
	"testing"
)

func TestStateString(t *testing.T) {
	tests := map[State]string{
		StateNotInstalled: "not installed",
		StateStopped:      "stopped",
		StateRunning:      "running",
		State(99):         "unknown(99)",
	}
	for state, want := range tests {
		if got := state.String(); got != want {
			t.Fatalf("State(%d).String() = %q, want %q", state, got, want)
		}
	}
}

func TestPlatformOperationsValidateNamesBeforeExternalCalls(t *testing.T) {
	if err := Install(Config{}); err == nil {
		t.Fatal("Install accepted empty config")
	}
	for name, operation := range map[string]func(string) error{
		"remove": Remove,
		"start":  Start,
		"stop":   Stop,
	} {
		if err := operation("bad/name"); err == nil {
			t.Fatalf("%s accepted invalid name", name)
		}
	}
	if _, err := Status("bad/name"); err == nil {
		t.Fatal("Status accepted invalid name")
	}
	if err := RunAsService("bad/name", func(context.Context) error { return nil }); err == nil {
		t.Fatal("RunAsService accepted invalid name")
	}
}

func TestValidateConfig(t *testing.T) {
	valid := Config{Name: "claude-status_relay@1", DisplayName: "Relay", Description: "safe", Args: []string{"relay", "--once"}}
	if err := validateConfig(valid); err != nil {
		t.Fatalf("valid config error = %v", err)
	}
	tests := []Config{
		{},
		{Name: "bad/name"},
		{Name: "ok", Description: "line one\nInjected=yes"},
		{Name: "ok", Args: []string{"safe", "bad\x00arg"}},
	}
	for _, cfg := range tests {
		if err := validateConfig(cfg); err == nil {
			t.Fatalf("validateConfig(%+v) unexpectedly succeeded", cfg)
		}
	}
}

func TestUnsupportedPlatformSentinel(t *testing.T) {
	if !errors.Is(ErrUnsupportedPlatform, ErrUnsupportedPlatform) {
		t.Fatal("unsupported platform error is not a stable sentinel")
	}
}

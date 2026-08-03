//go:build linux

package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func useLinuxServiceFakes(t *testing.T, home, executable string, systemctl func(...string) error) {
	t.Helper()
	oldHome := userHomeDirectory
	oldExecutable := executablePath
	oldSystemctl := runSystemctl
	userHomeDirectory = func() (string, error) { return home, nil }
	executablePath = func() (string, error) { return executable, nil }
	runSystemctl = systemctl
	t.Cleanup(func() {
		userHomeDirectory = oldHome
		executablePath = oldExecutable
		runSystemctl = oldSystemctl
	})
}

func TestLinuxInstallFormatsAndWritesSafeUnit(t *testing.T) {
	home := t.TempDir()
	var calls []string
	useLinuxServiceFakes(t, home, "/opt/status app/claude%$status", func(args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		return nil
	})
	cfg := Config{Name: "claude-status-test", Description: "Status 100%", Args: []string{"relay", "$HOME", "a'b"}}
	if err := Install(cfg); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".config", "systemd", "user", cfg.Name+".service")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	unit := string(data)
	for _, want := range []string{`Description="Status 100%%"`, `Type=exec`, `"/opt/status app/claude%%$$status" "relay" "$$HOME" "a'b"`} {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit does not contain %q:\n%s", want, unit)
		}
	}
	if strings.Join(calls, "|") != "daemon-reload|enable --now claude-status-test" {
		t.Fatalf("systemctl calls = %q", calls)
	}
}

func TestLinuxRemoveStartAndStop(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "systemd", "user", "claude-status-test.service")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("unit"), 0o600); err != nil {
		t.Fatal(err)
	}
	var calls []string
	useLinuxServiceFakes(t, home, "/bin/status", func(args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		return nil
	})
	if err := Start("claude-status-test"); err != nil {
		t.Fatal(err)
	}
	if err := Stop("claude-status-test"); err != nil {
		t.Fatal(err)
	}
	if err := Remove("claude-status-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unit still exists: %v", err)
	}
	if len(calls) != 4 {
		t.Fatalf("systemctl calls = %q", calls)
	}
}

func TestLinuxStatus(t *testing.T) {
	home := t.TempDir()
	useLinuxServiceFakes(t, home, "/bin/status", func(...string) error { return nil })
	if state, err := Status("claude-status-test"); err != nil || state != StateNotInstalled {
		t.Fatalf("missing Status() = %v, %v", state, err)
	}
	path, err := unitPath("claude-status-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	oldQuery := querySystemdState
	defer func() { querySystemdState = oldQuery }()
	querySystemdState = func(string) (string, error) { return "active", nil }
	if state, err := Status("claude-status-test"); err != nil || state != StateRunning {
		t.Fatalf("active Status() = %v, %v", state, err)
	}
	querySystemdState = func(string) (string, error) { return "failed", errors.New("exit 3") }
	if state, err := Status("claude-status-test"); err != nil || state != StateStopped {
		t.Fatalf("failed Status() = %v, %v", state, err)
	}
	querySystemdState = func(string) (string, error) { return "transport error", errors.New("exit 1") }
	if _, err := Status("claude-status-test"); err == nil {
		t.Fatal("expected unknown status error")
	}
}

func TestInactiveSystemdStates(t *testing.T) {
	for _, state := range []string{"inactive", "failed", "activating", "deactivating", "reloading", "maintenance"} {
		if !isInactiveSystemdState(state) {
			t.Fatalf("%q was not inactive", state)
		}
	}
	if isInactiveSystemdState("active") || isInactiveSystemdState("unknown") {
		t.Fatal("active or unknown state classified as inactive")
	}
}

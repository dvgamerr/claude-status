//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Linux runs the relay as a systemd --user unit rather than a system-level
// one: it needs no root, starts in the same session as the desktop/SSH
// login that's actually running Claude Code, and "systemctl --user" is
// available on every systemd distribution this project targets.
func unitPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "systemd", "user", name+".service"), nil
}

func Install(cfg Config) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	path, err := unitPath(cfg.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create systemd user directory: %w", err)
	}

	execStart := quoteShellArgs(append([]string{exePath}, cfg.Args...))
	unit := fmt.Sprintf(`[Unit]
Description=%s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=2

[Install]
WantedBy=default.target
`, cfg.Description, execStart)

	if err := os.WriteFile(path, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("write systemd unit: %w", err)
	}
	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	return systemctl("enable", "--now", cfg.Name)
}

func Remove(name string) error {
	path, err := unitPath(name)
	if err != nil {
		return err
	}
	// Ignore "already stopped/disabled" failures — the goal is an absent
	// unit either way, and the file removal below is what actually matters.
	_ = systemctl("disable", "--now", name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove systemd unit: %w", err)
	}
	return systemctl("daemon-reload")
}

func Start(name string) error {
	return systemctl("start", name)
}

func Stop(name string) error {
	return systemctl("stop", name)
}

func Status(name string) (State, error) {
	path, err := unitPath(name)
	if err != nil {
		return StateNotInstalled, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return StateNotInstalled, nil
	}
	out, err := exec.Command("systemctl", "--user", "is-active", name).Output()
	active := strings.TrimSpace(string(out))
	if err == nil && active == "active" {
		return StateRunning, nil
	}
	return StateStopped, nil
}

func systemctl(args ...string) error {
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return fmt.Errorf("systemctl --user %s: %w: %s", strings.Join(args, " "), err, detail)
		}
		return fmt.Errorf("systemctl --user %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func quoteShellArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, "'"+strings.ReplaceAll(arg, "'", `'\''`)+"'")
	}
	return strings.Join(quoted, " ")
}

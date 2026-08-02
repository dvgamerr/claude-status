package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dvgamerr/claude-status/internal/logging"
)

// tty1UnitName is the systemd unit that owns /dev/tty1 on the Pi — see
// configs/claude-status-tty1.service, which `pi install` supersedes with a
// version whose paths/user are resolved at install time instead of
// hardcoded to /home/pi.
const tty1UnitName = "claude-status-tty1.service"

// runPi is the display-side counterpart to `service` (which sets up the
// relay on the machine running Claude Code/Codex): `claude-status pi
// install` sets up the framebuffer dashboard on the Raspberry Pi itself.
func runPi(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: claude-status pi <install> [flags]")
		return 2
	}
	switch args[0] {
	case "install":
		return runPiInstall(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		fmt.Fprintln(stdout, "Usage: claude-status pi install [flags]")
		return 0
	default:
		logger := logging.New(stderr, "pi")
		logger.Error().Str("subcommand", args[0]).Msg("unknown subcommand")
		return 2
	}
}

// runPiInstall writes and enables the tty1 framebuffer service, pointing it
// at wherever this binary actually is — which lets `go install .../cmd/
// claude-status@latest` followed by `claude-status pi install` work with no
// git checkout at all, unlike the old scripts/install.sh + manual
// `sudo systemctl enable` flow.
func runPiInstall(args []string, stdout, stderr io.Writer) int {
	logger := logging.New(stderr, "pi install")
	if runtime.GOOS != "linux" {
		logger.Error().Msg("only supported on Linux (Raspberry Pi OS)")
		return 1
	}

	flags := newCommandFlagSet("pi install", "Usage: claude-status pi install [--user NAME] [--refresh 66ms] [--framebuffer /dev/fb0] [--tty /dev/tty1] [--touch-device /dev/input/event0]", stderr)
	userName := flags.String("user", "", "user the dashboard service runs as (default: $SUDO_USER, or the current user)")
	refresh := flags.String("refresh", "66ms", "frame refresh interval")
	framebufferPath := flags.String("framebuffer", "/dev/fb0", "Linux framebuffer device")
	ttyPath := flags.String("tty", "/dev/tty1", "virtual console switched to graphics mode")
	touchDevice := flags.String("touch-device", "/dev/input/event0", "evdev device for touch feedback (empty disables it)")
	if exitCode, parsed := parseCommandFlags(flags, args); !parsed {
		return exitCode
	}
	if flags.NArg() != 0 {
		logger.Error().Msg("unexpected positional arguments")
		return 2
	}

	exePath, err := os.Executable()
	if err != nil {
		logger.Error().Err(err).Msg("resolve executable path")
		return 1
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}

	// Writing the unit file and calling systemctl both need root; the
	// tty1 device itself is owned by the target user via SupplementaryGroups,
	// not by running the dashboard process as root.
	if os.Geteuid() != 0 {
		return reExecWithSudo(exePath, args, stderr)
	}

	targetUser := strings.TrimSpace(*userName)
	if targetUser == "" {
		targetUser = os.Getenv("SUDO_USER")
	}
	if targetUser == "" {
		targetUser = "pi"
	}
	account, err := user.Lookup(targetUser)
	if err != nil {
		logger.Error().Err(err).Str("user", targetUser).Msg("look up user")
		return 1
	}

	execStart := fmt.Sprintf("%s gfx --refresh %s --framebuffer %s --tty %s", exePath, *refresh, *framebufferPath, *ttyPath)
	if strings.TrimSpace(*touchDevice) != "" {
		execStart += " --touch-device " + *touchDevice
	}

	unit := fmt.Sprintf(`[Unit]
Description=AI Status 800x480 framebuffer dashboard
After=local-fs.target systemd-user-sessions.service
Conflicts=getty@%s.service

[Service]
Type=simple
User=%s
Group=%s
SupplementaryGroups=video input
WorkingDirectory=%s
Environment=HOME=%s
ExecStartPre=/usr/bin/test -x %s
ExecStart=%s
Restart=always
RestartSec=2
StandardInput=tty
StandardOutput=journal
StandardError=journal
TTYPath=%s
TTYReset=yes
TTYVHangup=yes
TTYVTDisallocate=yes

[Install]
WantedBy=multi-user.target
`, filepath.Base(*ttyPath), account.Username, account.Username, account.HomeDir, account.HomeDir, exePath, execStart, *ttyPath)

	unitDestination := filepath.Join("/etc/systemd/system", tty1UnitName)
	if err := os.WriteFile(unitDestination, []byte(unit), 0o644); err != nil {
		logger.Error().Err(err).Str("path", unitDestination).Msg("write unit file")
		return 1
	}
	if err := runSystemCommand("systemctl", "daemon-reload"); err != nil {
		logger.Error().Err(err).Msg("systemctl daemon-reload")
		return 1
	}
	if err := runSystemCommand("systemctl", "enable", "--now", tty1UnitName); err != nil {
		logger.Error().Err(err).Msg("systemctl enable")
		return 1
	}
	fmt.Fprintf(stdout, "installed and started %s (binary: %s, user: %s)\n", tty1UnitName, exePath, account.Username)
	return 0
}

// reExecWithSudo re-invokes this same subcommand under sudo so the user can
// run one command (`claude-status pi install`) instead of having to
// remember to prefix it themselves.
func reExecWithSudo(exePath string, args []string, stderr io.Writer) int {
	sudoArgs := append([]string{exePath, "pi", "install"}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		logger := logging.New(stderr, "pi install")
		logger.Error().Err(err).Msg("re-exec via sudo")
		return 1
	}
	return 0
}

func runSystemCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, detail)
		}
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

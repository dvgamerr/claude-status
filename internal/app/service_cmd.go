package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/mirror"
	"github.com/dvgamerr/claude-status/internal/service"
	"github.com/dvgamerr/claude-status/internal/state"
)

// RelayServiceName identifies the background service across all three
// platforms: the Windows Service name, the systemd --user unit name (sans
// ".service"), and the launchd LaunchAgent label.
const RelayServiceName = "claude-status-relay"

// runService is the one `claude-status service <verb>` entry point for
// Windows/Linux/macOS alike — see internal/service for how each OS's own
// service manager is driven underneath.
func runService(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: claude-status service <install|remove|start|stop|status> [flags]")
		return 2
	}
	switch args[0] {
	case "install":
		return runServiceInstall(args[1:], stderr)
	case "remove":
		if err := service.Remove(RelayServiceName); err != nil {
			fmt.Fprintf(stderr, "claude-status service remove: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "removed %s\n", RelayServiceName)
		return 0
	case "start":
		return runServiceControl(service.Start, "start", stdout, stderr)
	case "stop":
		return runServiceControl(service.Stop, "stop", stdout, stderr)
	case "status":
		state, err := service.Status(RelayServiceName)
		if err != nil {
			fmt.Fprintf(stderr, "claude-status service status: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s: %s\n", RelayServiceName, state)
		return 0
	case "help", "--help", "-h":
		fmt.Fprintln(stdout, "Usage: claude-status service <install|remove|start|stop|status> [flags]")
		return 0
	default:
		fmt.Fprintf(stderr, "claude-status service: unknown subcommand %q\n", args[0])
		return 2
	}
}

// runServiceInstall registers (or reconfigures, if already installed) the
// relay as a background service and starts it — the cross-platform
// replacement for install-windows.ps1's Scheduled Task registration.
func runServiceInstall(args []string, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status service install: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("service install", flag.ContinueOnError)
	flags.SetOutput(stderr)
	mirrorSSH := flags.String("mirror-ssh", "", "SSH host that receives sanitized snapshots")
	remoteBinary := flags.String("remote-bin", mirror.DefaultRemoteBinary, "claude-status binary on the SSH mirror")
	refresh := flags.Duration("refresh", time.Second, "interval between local snapshot checks")
	stateDir := flags.String("state-dir", defaultDir, "directory containing sanitized snapshots")
	logFile := flags.String("log-file", filepath.Join(defaultDir, "relay.log"), "relay diagnostics log file")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status service install --mirror-ssh HOST [--remote-bin PATH] [--refresh 1s] [--state-dir DIR] [--log-file FILE]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status service install: unexpected positional arguments")
		return 2
	}
	if strings.TrimSpace(*mirrorSSH) == "" {
		fmt.Fprintln(stderr, "claude-status service install: --mirror-ssh is required")
		return 2
	}

	cfg := service.Config{
		Name:        RelayServiceName,
		DisplayName: "Claude Status Relay",
		Description: "Mirrors sanitized Claude/Codex usage snapshots to a remote dashboard over SSH.",
		Args: []string{
			"relay",
			"--mirror-ssh", *mirrorSSH,
			"--remote-bin", *remoteBinary,
			"--refresh", refresh.String(),
			"--state-dir", *stateDir,
			"--log-file", *logFile,
		},
	}
	if err := service.Install(cfg); err != nil {
		fmt.Fprintf(stderr, "claude-status service install: %v\n", err)
		return 1
	}
	fmt.Fprintf(stderr, "installed and started %s (log: %s)\n", RelayServiceName, *logFile)
	return 0
}

func runServiceControl(action func(string) error, verb string, stdout, stderr io.Writer) int {
	if err := action(RelayServiceName); err != nil {
		fmt.Fprintf(stderr, "claude-status service %s: %v\n", verb, err)
		return 1
	}
	state, err := service.Status(RelayServiceName)
	if err != nil {
		fmt.Fprintf(stdout, "%s: %s requested\n", RelayServiceName, verb)
		return 0
	}
	fmt.Fprintf(stdout, "%s: %s\n", RelayServiceName, state)
	return 0
}

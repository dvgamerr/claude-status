package app

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/codex"
	"github.com/dvgamerr/claude-status/internal/dashboard"
	"github.com/dvgamerr/claude-status/internal/ingest"
	"github.com/dvgamerr/claude-status/internal/mirror"
	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/state"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "ingest":
		return runIngest(ctx, args[1:], stdin, stdout, stderr)
	case "codex-notify":
		return runCodexNotify(ctx, args[1:], stderr)
	case "import":
		return runImport(args[1:], stdin, stderr)
	case "tui":
		return runTUI(ctx, args[1:], stdin, stdout, stderr)
	case "version", "--version", "-version":
		fmt.Fprintf(stdout, "claude-status %s (commit %s, built %s)\n", Version, Commit, Date)
		return 0
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "claude-status: unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runIngest(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status ingest: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("ingest", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory used for sanitized snapshots")
	mirrorSSH := flags.String("mirror-ssh", "", "SSH host that receives the sanitized snapshot")
	remoteBinary := flags.String("remote-bin", mirror.DefaultRemoteBinary, "claude-status binary on the SSH mirror")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status ingest [--state-dir DIR] [--mirror-ssh HOST]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status ingest: unexpected positional arguments")
		return 2
	}
	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status ingest: %v\n", err)
		return 1
	}
	snapshot, err := ingest.Run(stdin, stdout, store, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "claude-status ingest: %v\n", err)
		return 1
	}
	if err := mirrorIfConfigured(ctx, *mirrorSSH, *remoteBinary, snapshot); err != nil {
		fmt.Fprintf(stderr, "claude-status ingest: warning: %v\n", err)
	}
	return 0
}

func runImport(args []string, stdin io.Reader, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status import: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory used for sanitized snapshots")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status import [--state-dir DIR]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status import: unexpected positional arguments")
		return 2
	}
	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status import: %v\n", err)
		return 1
	}
	snapshot, err := state.DecodeSnapshot(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status import: %v\n", err)
		return 1
	}
	if err := store.Save(snapshot); err != nil {
		fmt.Fprintf(stderr, "claude-status import: %v\n", err)
		return 1
	}
	return 0
}

type stringList []string

func (values *stringList) String() string { return strings.Join(*values, ",") }
func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func runCodexNotify(ctx context.Context, args []string, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status codex-notify: %v\n", err)
		return 1
	}
	defaultCodexHome, err := codex.DefaultHome()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status codex-notify: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("codex-notify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory used for sanitized snapshots")
	codexHome := flags.String("codex-home", defaultCodexHome, "Codex home containing session rollouts")
	mirrorSSH := flags.String("mirror-ssh", "", "SSH host that receives the sanitized snapshot")
	remoteBinary := flags.String("remote-bin", mirror.DefaultRemoteBinary, "claude-status binary on the SSH mirror")
	forward := flags.String("forward", "", "existing notifier executable to preserve")
	var forwardArgs stringList
	flags.Var(&forwardArgs, "forward-arg", "argument for the existing notifier (repeatable)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status codex-notify [flags] NOTIFICATION_JSON")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "claude-status codex-notify: expected one Codex notification JSON argument")
		return 2
	}
	rawNotification := flags.Arg(0)
	if *forward != "" {
		if err := forwardNotification(ctx, *forward, forwardArgs, rawNotification); err != nil {
			fmt.Fprintf(stderr, "claude-status codex-notify: warning: %v\n", err)
		}
	}
	notification, err := codex.DecodeNotification(rawNotification)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status codex-notify: %v\n", err)
		return 1
	}
	snapshot, err := codex.SnapshotFromNotification(notification, *codexHome, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "claude-status codex-notify: %v\n", err)
		return 1
	}
	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status codex-notify: %v\n", err)
		return 1
	}
	if err := store.Save(snapshot); err != nil {
		fmt.Fprintf(stderr, "claude-status codex-notify: %v\n", err)
		return 1
	}
	if err := mirrorIfConfigured(ctx, *mirrorSSH, *remoteBinary, snapshot); err != nil {
		fmt.Fprintf(stderr, "claude-status codex-notify: warning: %v\n", err)
	}
	return 0
}

func mirrorIfConfigured(ctx context.Context, host, remoteBinary string, snapshot model.Snapshot) error {
	if strings.TrimSpace(host) == "" {
		return nil
	}
	return mirror.SSH(ctx, host, remoteBinary, snapshot)
}

func forwardNotification(ctx context.Context, program string, args []string, payload string) error {
	forwardCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	commandArgs := append(append([]string{}, args...), payload)
	command := exec.CommandContext(forwardCtx, program, commandArgs...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return fmt.Errorf("forward Codex notification: %w: %s", err, detail)
		}
		return fmt.Errorf("forward Codex notification: %w", err)
	}
	return nil
}

func runTUI(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status tui: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("tui", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory containing sanitized snapshots")
	sessionID := flags.String("session", "", "initial session ID (defaults to most recent)")
	refresh := flags.Duration("refresh", time.Second, "dashboard refresh interval")
	staleAfter := flags.Duration("stale-after", 15*time.Second, "age at which a snapshot is marked stale")
	inline := flags.Bool("inline", false, "render without the terminal alternate screen")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status tui [--state-dir DIR] [--session ID] [--refresh 1s] [--stale-after 15s] [--inline]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status tui: unexpected positional arguments")
		return 2
	}
	if *refresh < 250*time.Millisecond {
		fmt.Fprintln(stderr, "claude-status tui: --refresh must be at least 250ms")
		return 2
	}
	if *staleAfter <= 0 {
		fmt.Fprintln(stderr, "claude-status tui: --stale-after must be greater than zero")
		return 2
	}
	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status tui: %v\n", err)
		return 1
	}
	config := dashboard.Config{
		RefreshInterval: *refresh,
		StaleAfter:      *staleAfter,
		InitialSession:  strings.TrimSpace(*sessionID),
		Inline:          *inline,
	}
	if err := dashboard.Run(ctx, stdin, stdout, store, systeminfo.NewReader("/"), config); err != nil {
		fmt.Fprintf(stderr, "claude-status tui: %v\n", err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `AI Usage Terminal for Raspberry Pi

Usage:
  claude-status ingest [flags]       Read Claude Code statusLine JSON from stdin
  claude-status codex-notify [flags] Read a Codex turn-complete notification
  claude-status import [flags]       Import one sanitized snapshot
  claude-status tui [flags]          Open the full-screen dashboard
  claude-status version              Print build information

Run "claude-status <command> --help" for command flags.`)
}

package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/dashboard"
	"github.com/dvgamerr/claude-status/internal/ingest"
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
		return runIngest(args[1:], stdin, stdout, stderr)
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

func runIngest(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status ingest: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("ingest", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory used for sanitized snapshots")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status ingest [--state-dir DIR]")
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
	if err := ingest.Run(stdin, stdout, store, time.Now()); err != nil {
		fmt.Fprintf(stderr, "claude-status ingest: %v\n", err)
		return 1
	}
	return 0
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
	fmt.Fprintln(w, `Claude Usage Terminal for Raspberry Pi

Usage:
  claude-status ingest [flags]  Read Claude Code statusLine JSON from stdin
  claude-status tui [flags]     Open the full-screen dashboard
  claude-status version         Print build information

Run "claude-status <command> --help" for command flags.`)
}

package app

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/activity"
	"github.com/dvgamerr/claude-status/internal/codex"
	"github.com/dvgamerr/claude-status/internal/dashboard"
	"github.com/dvgamerr/claude-status/internal/framebuffer"
	"github.com/dvgamerr/claude-status/internal/ingest"
	"github.com/dvgamerr/claude-status/internal/mirror"
	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/pixelui"
	"github.com/dvgamerr/claude-status/internal/relay"
	"github.com/dvgamerr/claude-status/internal/state"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
	"github.com/dvgamerr/claude-status/internal/touch"
	"github.com/dvgamerr/claude-status/internal/usage"
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
	case "activity":
		return runActivity(args[1:], stdin, stderr)
	case "usage":
		return runUsage(args[1:], stderr)
	case "codex-notify":
		return runCodexNotify(ctx, args[1:], stderr)
	case "import":
		return runImport(args[1:], stdin, stderr)
	case "relay":
		return runRelay(ctx, args[1:], stderr)
	case "tui":
		return runTUI(ctx, args[1:], stdin, stdout, stderr)
	case "gfx":
		return runGFX(ctx, args[1:], stderr)
	case "preview":
		return runPreview(args[1:], stderr)
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
	_, err = ingest.Run(stdin, stdout, store, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "claude-status ingest: %v\n", err)
		return 1
	}
	return 0
}

// runActivity records a Claude Code hook event (Notification, PreToolUse,
// Stop, UserPromptSubmit) as a working/idle/waiting-approval indicator. It
// always exits 0: some of these hook events (PreToolUse) can block the tool
// call if the hook exits non-zero, and this side channel must never do that.
func runActivity(args []string, stdin io.Reader, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status activity: %v\n", err)
		return 0
	}
	flags := flag.NewFlagSet("activity", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory used for sanitized snapshots")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status activity [--state-dir DIR]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status activity: unexpected positional arguments")
		return 0
	}
	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status activity: %v\n", err)
		return 0
	}
	_, _, err = activity.Run(stdin, store, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "claude-status activity: %v\n", err)
		return 0
	}
	return 0
}

// runUsage manually merges the two rate-limit percentages into the target
// session's stored snapshot, for interfaces (this project has only ever
// seen the VS Code extension chat panel) where statusLine never fires so
// claude-status never sees real numbers on its own.
func runUsage(args []string, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status usage: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("usage", flag.ContinueOnError)
	flags.SetOutput(stderr)
	fiveHour := flags.Float64("five-hour", -1, "5-hour rate limit used percentage (0-100), required")
	sevenDay := flags.Float64("seven-day", -1, "7-day rate limit used percentage (0-100), required")
	fiveHourReset := flags.Duration("five-hour-reset", 5*time.Hour, "time until the 5-hour window resets")
	sevenDayReset := flags.Duration("seven-day-reset", 7*24*time.Hour, "time until the 7-day window resets")
	sessionID := flags.String("session", "", "session ID to update (defaults to the most recently updated session)")
	stateDir := flags.String("state-dir", defaultDir, "directory used for sanitized snapshots")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status usage --five-hour PCT --seven-day PCT [--five-hour-reset 5h] [--seven-day-reset 168h] [--session ID] [--state-dir DIR]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status usage: unexpected positional arguments")
		return 2
	}
	if *fiveHour < 0 || *sevenDay < 0 {
		fmt.Fprintln(stderr, "claude-status usage: --five-hour and --seven-day are required")
		return 2
	}
	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status usage: %v\n", err)
		return 1
	}
	_, err = usage.Run(store, *sessionID, *fiveHour, *sevenDay, *fiveHourReset, *sevenDayReset, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "claude-status usage: %v\n", err)
		return 1
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
	return 0
}

func runRelay(ctx context.Context, args []string, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status relay: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("relay", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory containing sanitized snapshots")
	mirrorSSH := flags.String("mirror-ssh", "", "SSH host that receives sanitized snapshots")
	remoteBinary := flags.String("remote-bin", mirror.DefaultRemoteBinary, "claude-status binary on the SSH mirror")
	refresh := flags.Duration("refresh", time.Second, "interval between local snapshot checks")
	once := flags.Bool("once", false, "send pending snapshots once and exit")
	logFile := flags.String("log-file", "", "append relay diagnostics to this file")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status relay --mirror-ssh HOST [--refresh 1s] [--once] [--log-file FILE]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status relay: unexpected positional arguments")
		return 2
	}
	if strings.TrimSpace(*mirrorSSH) == "" {
		fmt.Fprintln(stderr, "claude-status relay: --mirror-ssh is required")
		return 2
	}
	if *refresh < 100*time.Millisecond {
		fmt.Fprintln(stderr, "claude-status relay: --refresh must be at least 100ms")
		return 2
	}

	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status relay: %v\n", err)
		return 1
	}
	logOutput := stderr
	var file *os.File
	if strings.TrimSpace(*logFile) != "" {
		if err := os.MkdirAll(filepath.Dir(*logFile), 0o700); err != nil {
			fmt.Fprintf(stderr, "claude-status relay: create log directory: %v\n", err)
			return 1
		}
		file, err = os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			fmt.Fprintf(stderr, "claude-status relay: open log file: %v\n", err)
			return 1
		}
		defer file.Close()
		logOutput = io.MultiWriter(stderr, file)
	}
	logger := log.New(logOutput, "claude-status relay: ", log.LstdFlags|log.Lmicroseconds)
	worker, err := relay.New(store, func(sendCtx context.Context, snapshot model.Snapshot) error {
		return mirror.SSH(sendCtx, *mirrorSSH, *remoteBinary, snapshot)
	}, logger.Printf)
	if err != nil {
		logger.Printf("%v", err)
		return 1
	}

	ticker := time.NewTicker(*refresh)
	defer ticker.Stop()
	for {
		syncErr := worker.Sync(ctx)
		if *once {
			if syncErr != nil {
				return 1
			}
			return 0
		}
		select {
		case <-ctx.Done():
			return 0
		case <-ticker.C:
		}
	}
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

func runGFX(ctx context.Context, args []string, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status gfx: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("gfx", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory containing sanitized snapshots")
	refresh := flags.Duration("refresh", time.Second/15, "frame refresh interval (default ~15fps)")
	staleAfter := flags.Duration("stale-after", 15*time.Second, "age at which a snapshot is marked stale")
	framebufferPath := flags.String("framebuffer", "/dev/fb0", "Linux framebuffer device")
	ttyPath := flags.String("tty", "/dev/tty1", "virtual console switched to graphics mode")
	touchDevice := flags.String("touch-device", "/dev/input/event0", "evdev device for the touchscreen (empty disables touch feedback)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status gfx [--state-dir DIR] [--refresh 66ms] [--framebuffer /dev/fb0] [--tty /dev/tty1] [--touch-device /dev/input/event0]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status gfx: unexpected positional arguments")
		return 2
	}
	if *refresh < 20*time.Millisecond {
		fmt.Fprintln(stderr, "claude-status gfx: --refresh must be at least 20ms")
		return 2
	}
	if *staleAfter <= 0 {
		fmt.Fprintln(stderr, "claude-status gfx: --stale-after must be greater than zero")
		return 2
	}
	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status gfx: %v\n", err)
		return 1
	}
	renderer, err := pixelui.NewRenderer()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status gfx: %v\n", err)
		return 1
	}
	screen, err := framebuffer.Open(*framebufferPath, *ttyPath)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status gfx: %v\n", err)
		return 1
	}
	defer func() {
		if err := screen.Close(); err != nil {
			fmt.Fprintf(stderr, "claude-status gfx: close display: %v\n", err)
		}
	}()
	if size := screen.Size(); size.X != pixelui.Width || size.Y != pixelui.Height {
		fmt.Fprintf(stderr, "claude-status gfx: display is %dx%d; expected %dx%d\n", size.X, size.Y, pixelui.Width, pixelui.Height)
		return 1
	}
	var touches <-chan touch.Point
	if strings.TrimSpace(*touchDevice) != "" {
		opened, err := touch.Watch(ctx, *touchDevice)
		if err != nil {
			// Touch feedback is a nice-to-have; the dashboard's real job
			// (showing usage) must keep working without it.
			fmt.Fprintf(stderr, "claude-status gfx: warning: touch input disabled: %v\n", err)
		} else {
			touches = opened
		}
	}

	config := pixelui.RunConfig{RefreshInterval: *refresh, StaleAfter: *staleAfter}
	if err := pixelui.Run(ctx, store, systeminfo.NewReader("/"), screen, renderer, config, touches); err != nil {
		fmt.Fprintf(stderr, "claude-status gfx: %v\n", err)
		return 1
	}
	return 0
}

func runPreview(args []string, stderr io.Writer) int {
	defaultDir, err := state.DefaultDir()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status preview: %v\n", err)
		return 1
	}
	flags := flag.NewFlagSet("preview", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateDir := flags.String("state-dir", defaultDir, "directory containing sanitized snapshots")
	outputPath := flags.String("output", "pixel-dashboard-preview.png", "PNG output path")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: claude-status preview [--state-dir DIR] [--output dashboard.png]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "claude-status preview: unexpected positional arguments")
		return 2
	}
	store, err := state.New(*stateDir)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status preview: %v\n", err)
		return 1
	}
	snapshots, loadErr := store.LoadAll()
	claude, codex := pixelui.LatestProviders(snapshots)
	renderer, err := pixelui.NewRenderer()
	if err != nil {
		fmt.Fprintf(stderr, "claude-status preview: %v\n", err)
		return 1
	}
	frame := renderer.Render(pixelui.View{Claude: claude, Codex: codex, Now: time.Now(), StaleAfter: 15 * time.Second, SessionCount: len(snapshots), LoadError: loadErr})
	file, err := os.Create(*outputPath)
	if err != nil {
		fmt.Fprintf(stderr, "claude-status preview: create output: %v\n", err)
		return 1
	}
	if err := png.Encode(file, frame); err != nil {
		file.Close()
		fmt.Fprintf(stderr, "claude-status preview: encode PNG: %v\n", err)
		return 1
	}
	if err := file.Close(); err != nil {
		fmt.Fprintf(stderr, "claude-status preview: close output: %v\n", err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `AI Usage Terminal for Raspberry Pi

Usage:
  claude-status ingest [flags]       Read Claude Code statusLine JSON from stdin
  claude-status activity [flags]     Read a Claude Code hook event from stdin
  claude-status usage [flags]        Manually set 5h/7d rate-limit percentages
  claude-status codex-notify [flags] Read a Codex turn-complete notification
  claude-status import [flags]       Import one sanitized snapshot
  claude-status relay [flags]        Retry local snapshot delivery over SSH
  claude-status gfx [flags]          Render the 800x480 framebuffer dashboard
  claude-status preview [flags]      Save one framebuffer dashboard frame as PNG
  claude-status tui [flags]          Open the full-screen dashboard
  claude-status version              Print build information

Run "claude-status <command> --help" for command flags.`)
}

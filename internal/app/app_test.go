package app

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dvgamerr/claude-status/internal/state"
)

func TestRunIngestEndToEnd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	input := bytes.NewBufferString(`{
      "session_id":"cli-test",
      "model":{"display_name":"Opus"},
      "context_window":{"used_percentage":12},
      "rate_limits":{"five_hour":{"used_percentage":34}}
    }`)
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"ingest", "--state-dir", dir}, input, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Run() exit = %d, stderr = %s", exitCode, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "[Opus] 5h 34% · ctx 12%" {
		t.Fatalf("stdout = %q", got)
	}
	store, err := state.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest() error = %v", err)
	}
	if snapshot.Session.ID != "cli-test" {
		t.Fatalf("stored session = %q", snapshot.Session.ID)
	}
}

func TestRunRejectsInvalidInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"ingest", "--state-dir", t.TempDir()}, strings.NewReader("not-json"), &stdout, &stderr)
	if exitCode != 1 || !strings.Contains(stderr.String(), "decode statusLine JSON") {
		t.Fatalf("exit=%d stderr=%q", exitCode, stderr.String())
	}
}

func TestRunHelpAndUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exitCode := Run(context.Background(), nil, strings.NewReader(""), &stdout, &stderr); exitCode != 0 || !strings.Contains(stdout.String(), "claude-status ingest") {
		t.Fatalf("help: exit=%d stdout=%q", exitCode, stdout.String())
	}
	stdout.Reset()
	if exitCode := Run(context.Background(), []string{"wat"}, strings.NewReader(""), &stdout, &stderr); exitCode != 2 || !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("unknown command: exit=%d stderr=%q", exitCode, stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"version"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 || !strings.Contains(stdout.String(), "claude-status dev") || stderr.Len() != 0 {
		t.Fatalf("version: exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
}

func TestRunValidatesCommandFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		wantText string
	}{
		{name: "ingest help", args: []string{"ingest", "--help"}, wantExit: 0, wantText: "Usage: claude-status ingest"},
		{name: "ingest positional", args: []string{"ingest", "unexpected"}, wantExit: 2, wantText: "unexpected positional"},
		{name: "ingest empty state", args: []string{"ingest", "--state-dir", ""}, wantExit: 1, wantText: "state directory is empty"},
		{name: "tui help", args: []string{"tui", "--help"}, wantExit: 0, wantText: "Usage: claude-status tui"},
		{name: "tui positional", args: []string{"tui", "unexpected"}, wantExit: 2, wantText: "unexpected positional"},
		{name: "tui refresh too fast", args: []string{"tui", "--refresh", "100ms"}, wantExit: 2, wantText: "at least 250ms"},
		{name: "tui stale invalid", args: []string{"tui", "--stale-after", "0s"}, wantExit: 2, wantText: "greater than zero"},
		{name: "tui duration invalid", args: []string{"tui", "--refresh", "invalid"}, wantExit: 2, wantText: "invalid value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := Run(context.Background(), tt.args, strings.NewReader(""), &stdout, &stderr)
			combined := stdout.String() + stderr.String()
			if exitCode != tt.wantExit || !strings.Contains(combined, tt.wantText) {
				t.Fatalf("exit=%d output=%q, want exit=%d containing %q", exitCode, combined, tt.wantExit, tt.wantText)
			}
		})
	}
}

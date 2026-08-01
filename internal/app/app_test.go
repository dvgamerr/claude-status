package app

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
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

func TestRunUsageEndToEnd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	seed := bytes.NewBufferString(`{"session_id":"cli-test","model":{"display_name":"Sonnet 5"}}`)
	var seedOut, seedErr bytes.Buffer
	if exitCode := Run(context.Background(), []string{"ingest", "--state-dir", dir}, seed, &seedOut, &seedErr); exitCode != 0 {
		t.Fatalf("seed ingest exit = %d, stderr = %s", exitCode, seedErr.String())
	}

	var stdout, stderr bytes.Buffer
	args := []string{"usage", "--state-dir", dir, "--five-hour", "63", "--seven-day", "42"}
	if exitCode := Run(context.Background(), args, strings.NewReader(""), &stdout, &stderr); exitCode != 0 {
		t.Fatalf("Run() exit = %d, stderr = %s", exitCode, stderr.String())
	}

	store, err := state.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest() error = %v", err)
	}
	if snapshot.Model.DisplayName != "Sonnet 5" {
		t.Fatalf("usage clobbered model: %+v", snapshot.Model)
	}
	if got := *snapshot.RateLimits.FiveHour.UsedPercentage; got != 63 {
		t.Fatalf("FiveHour.UsedPercentage = %v", got)
	}
	if got := *snapshot.RateLimits.SevenDay.UsedPercentage; got != 42 {
		t.Fatalf("SevenDay.UsedPercentage = %v", got)
	}
}

func TestRunRejectsInvalidInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"ingest", "--state-dir", t.TempDir()}, strings.NewReader("not-json"), &stdout, &stderr)
	if exitCode != 1 || !strings.Contains(stderr.String(), "decode statusLine JSON") {
		t.Fatalf("exit=%d stderr=%q", exitCode, stderr.String())
	}
}

func TestRunImportEndToEnd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	want := model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion,
		CapturedAt:    time.Unix(123, 0).UTC(),
		Provider:      "codex",
		Session:       model.Session{ID: "import-test"},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"import", "--state-dir", dir}, bytes.NewReader(data), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Run(import) exit = %d, stderr = %q", exitCode, stderr.String())
	}
	store, err := state.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadLatest()
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "codex" || got.Session.ID != "import-test" {
		t.Fatalf("imported snapshot = %+v", got)
	}
}

func TestRunCodexNotifyEndToEnd(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "codex")
	stateDir := filepath.Join(root, "state")
	threadID := "thread-app-test"
	rollout := filepath.Join(home, "sessions", "rollout-"+threadID+".jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	content := `{"type":"turn_context","payload":{"model":"gpt-5.6-sol","effort":"high"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"output_tokens":20,"total_tokens":120},"model_context_window":1000},"rate_limits":{}}}`
	if err := os.WriteFile(rollout, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	payload := `{"type":"agent-turn-complete","thread-id":"` + threadID + `","input-messages":["private"]}`
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"codex-notify", "--state-dir", stateDir, "--codex-home", home, payload}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Run(codex-notify) exit = %d, stderr = %q", exitCode, stderr.String())
	}
	store, err := state.New(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.LoadLatest()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Provider != "codex" || snapshot.Model.ID != "gpt-5.6-sol" || snapshot.Context.UsedPercentage == nil || *snapshot.Context.UsedPercentage != 12 {
		t.Fatalf("Codex snapshot = %+v", snapshot)
	}
}

func TestRunPreviewWritesPNG(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "preview.png")
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"preview", "--state-dir", filepath.Join(root, "state"), "--output", output}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Run(preview) exit = %d, stderr = %q", exitCode, stderr.String())
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != 800 || config.Height != 480 {
		t.Fatalf("preview size = %dx%d", config.Width, config.Height)
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
		{name: "import help", args: []string{"import", "--help"}, wantExit: 0, wantText: "Usage: claude-status import"},
		{name: "import positional", args: []string{"import", "unexpected"}, wantExit: 2, wantText: "unexpected positional"},
		{name: "codex notify help", args: []string{"codex-notify", "--help"}, wantExit: 0, wantText: "Usage: claude-status codex-notify"},
		{name: "codex notify missing payload", args: []string{"codex-notify"}, wantExit: 2, wantText: "expected one"},
		{name: "tui help", args: []string{"tui", "--help"}, wantExit: 0, wantText: "Usage: claude-status tui"},
		{name: "tui positional", args: []string{"tui", "unexpected"}, wantExit: 2, wantText: "unexpected positional"},
		{name: "tui refresh too fast", args: []string{"tui", "--refresh", "100ms"}, wantExit: 2, wantText: "at least 250ms"},
		{name: "tui stale invalid", args: []string{"tui", "--stale-after", "0s"}, wantExit: 2, wantText: "greater than zero"},
		{name: "tui duration invalid", args: []string{"tui", "--refresh", "invalid"}, wantExit: 2, wantText: "invalid value"},
		{name: "gfx help", args: []string{"gfx", "--help"}, wantExit: 0, wantText: "Usage: claude-status gfx"},
		{name: "gfx positional", args: []string{"gfx", "unexpected"}, wantExit: 2, wantText: "unexpected positional"},
		{name: "gfx refresh too fast", args: []string{"gfx", "--refresh", "5ms"}, wantExit: 2, wantText: "at least 20ms"},
		{name: "preview help", args: []string{"preview", "--help"}, wantExit: 0, wantText: "Usage: claude-status preview"},
		{name: "preview positional", args: []string{"preview", "unexpected"}, wantExit: 2, wantText: "unexpected positional"},
		{name: "usage help", args: []string{"usage", "--help"}, wantExit: 0, wantText: "Usage: claude-status usage"},
		{name: "usage positional", args: []string{"usage", "--five-hour", "1", "--seven-day", "1", "unexpected"}, wantExit: 2, wantText: "unexpected positional"},
		{name: "usage missing flags", args: []string{"usage"}, wantExit: 2, wantText: "--five-hour and --seven-day are required"},
		{name: "usage missing snapshot", args: []string{"usage", "--state-dir", t.TempDir(), "--five-hour", "1", "--seven-day", "1"}, wantExit: 1, wantText: "load existing snapshot"},
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

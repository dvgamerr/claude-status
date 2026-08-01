package ingest

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/state"
)

func TestRunPersistsSanitizedSnapshotAndWritesStatusLine(t *testing.T) {
	store, err := state.New(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	input := strings.NewReader(`{
      "session_id":"ingest-test",
      "model":{"display_name":"Opus"},
      "context_window":{"used_percentage":42},
      "transcript_path":"/must/not/be/stored"
    }`)
	var output bytes.Buffer
	now := time.Date(2026, 7, 31, 14, 0, 0, 0, time.UTC)

	if _, err := Run(input, &output, store, now); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := strings.TrimSpace(output.String()); got != "[Opus] ctx 42%" {
		t.Fatalf("status line = %q", got)
	}
	snapshot, err := store.LoadLatest()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Session.ID != "ingest-test" || !snapshot.CapturedAt.Equal(now) {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}

func TestRunPreservesActivityAcrossStatusLineRefresh(t *testing.T) {
	store, err := state.New(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 31, 14, 0, 0, 0, time.UTC)
	first := strings.NewReader(`{"session_id":"activity-test","context_window":{"used_percentage":10}}`)
	if _, err := Run(first, &bytes.Buffer{}, store, now); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	snapshot, err := store.LoadSession("activity-test")
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Activity = model.Activity{State: model.ActivityWorking, UpdatedAt: now}
	if err := store.Save(snapshot); err != nil {
		t.Fatal(err)
	}

	second := strings.NewReader(`{"session_id":"activity-test","context_window":{"used_percentage":20}}`)
	if _, err := Run(second, &bytes.Buffer{}, store, now.Add(5*time.Second)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	after, err := store.LoadSession("activity-test")
	if err != nil {
		t.Fatal(err)
	}
	if after.Activity.State != model.ActivityWorking {
		t.Fatalf("Activity.State = %q, want %q (statusLine refresh must not erase it)", after.Activity.State, model.ActivityWorking)
	}
}

func TestRunReturnsDecodeAndOutputErrors(t *testing.T) {
	store, err := state.New(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(strings.NewReader("not-json"), &bytes.Buffer{}, store, time.Now()); err == nil {
		t.Fatal("Run() accepted invalid JSON")
	}

	valid := strings.NewReader(`{"session_id":"output-error"}`)
	if _, err := Run(valid, errorWriter{}, store, time.Now()); err == nil || !strings.Contains(err.Error(), "write status line") {
		t.Fatalf("Run() output error = %v", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

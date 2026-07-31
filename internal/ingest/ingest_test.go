package ingest

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	if err := Run(input, &output, store, now); err != nil {
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

func TestRunReturnsDecodeAndOutputErrors(t *testing.T) {
	store, err := state.New(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(strings.NewReader("not-json"), &bytes.Buffer{}, store, time.Now()); err == nil {
		t.Fatal("Run() accepted invalid JSON")
	}

	valid := strings.NewReader(`{"session_id":"output-error"}`)
	if err := Run(valid, errorWriter{}, store, time.Now()); err == nil || !strings.Contains(err.Error(), "write status line") {
		t.Fatalf("Run() output error = %v", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

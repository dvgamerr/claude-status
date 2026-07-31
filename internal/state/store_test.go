package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
)

func TestSaveAndLoadMultipleSessions(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "state"))
	if err != nil {
		t.Fatal(err)
	}
	older := sampleSnapshot("session/with/unsafe/path", time.Unix(100, 0))
	newer := sampleSnapshot("second", time.Unix(200, 0))
	if err := store.Save(older); err != nil {
		t.Fatalf("Save(older) error = %v", err)
	}
	if err := store.Save(newer); err != nil {
		t.Fatalf("Save(newer) error = %v", err)
	}

	snapshots, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("LoadAll() returned %d snapshots, want 2", len(snapshots))
	}
	if snapshots[0].Session.ID != "second" || snapshots[1].Session.ID != older.Session.ID {
		t.Fatalf("snapshots are not newest-first: %+v", snapshots)
	}
	latest, err := store.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest() error = %v", err)
	}
	if latest.Session.ID != "second" {
		t.Fatalf("latest session = %q, want second", latest.Session.ID)
	}

	entries, err := os.ReadDir(filepath.Join(store.Dir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "session") || strings.Contains(entry.Name(), "unsafe") {
			t.Fatalf("session filename leaked or trusted the ID: %q", entry.Name())
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(store.Dir(), latestFile))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("latest.json permission = %o, want 600", got)
		}
	}
}

func TestSaveOverwritesSessionAtomically(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	first := sampleSnapshot("same", time.Unix(100, 0))
	second := sampleSnapshot("same", time.Unix(200, 0))
	if err := store.Save(first); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(second); err != nil {
		t.Fatal(err)
	}
	snapshots, err := store.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || !snapshots[0].CapturedAt.Equal(second.CapturedAt) {
		t.Fatalf("unexpected overwritten state: %+v", snapshots)
	}
	leftovers, err := filepath.Glob(filepath.Join(store.Dir(), "sessions", ".snapshot-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temporary files remain: %v", leftovers)
	}
}

func TestLoadAllSkipsCorruptFiles(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(sampleSnapshot("valid", time.Now())); err != nil {
		t.Fatal(err)
	}
	corrupt := filepath.Join(store.Dir(), "sessions", "corrupt.json")
	if err := os.WriteFile(corrupt, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshots, err := store.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || snapshots[0].Session.ID != "valid" {
		t.Fatalf("unexpected snapshots: %+v", snapshots)
	}
}

func TestDefaultDirHonorsEnvironment(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom")
	t.Setenv("CLAUDE_STATUS_STATE_DIR", want)
	got, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("DefaultDir() = %q, want %q", got, want)
	}
}

func sampleSnapshot(id string, capturedAt time.Time) model.Snapshot {
	return model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion,
		CapturedAt:    capturedAt.UTC(),
		Session:       model.Session{ID: id},
	}
}

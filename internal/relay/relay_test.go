package relay

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/state"
)

func TestSyncSendsLatestProviderSnapshotsOldestFirst(t *testing.T) {
	store := testStore(t)
	now := time.Now()
	saveSnapshot(t, store, "claude", "claude-old", now.Add(-3*time.Minute), "idle")
	saveSnapshot(t, store, "claude", "claude-new", now.Add(-time.Minute), "working")
	saveSnapshot(t, store, "codex", "codex-new", now, "")

	var got []string
	relay, err := New(store, func(_ context.Context, snapshot model.Snapshot) error {
		got = append(got, snapshot.Session.ID)
		return nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := relay.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "claude-new,codex-new" {
		t.Fatalf("sent sessions = %v", got)
	}
}

func TestSyncSkipsDeliveredContentAndSendsChangedActivity(t *testing.T) {
	store := testStore(t)
	now := time.Now()
	snapshot := saveSnapshot(t, store, "claude", "current", now, "idle")

	calls := 0
	var logs []string
	relay, err := New(store, func(_ context.Context, snapshot model.Snapshot) error {
		calls++
		return nil
	}, func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := relay.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := relay.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("unchanged snapshot sent %d times", calls)
	}

	snapshot.Activity = model.Activity{State: model.ActivityWorking, UpdatedAt: now.Add(time.Second)}
	if err := store.Save(snapshot); err != nil {
		t.Fatal(err)
	}
	if err := relay.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("changed snapshot left call count at %d", calls)
	}
	wantLog := fmt.Sprintf("mirrored claude snapshot captured at %s", now.Format(time.RFC3339Nano))
	if len(logs) != 1 || logs[0] != wantLog {
		t.Fatalf("successful refreshes logged repeatedly: %v", logs)
	}
}

func TestSyncRetriesFailureAndSuppressesRepeatedErrorLog(t *testing.T) {
	store := testStore(t)
	saveSnapshot(t, store, "claude", "current", time.Now(), "working")

	calls := 0
	var logs []string
	relay, err := New(store, func(_ context.Context, snapshot model.Snapshot) error {
		calls++
		if calls < 3 {
			return errors.New("network down")
		}
		return nil
	}, func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		_ = relay.Sync(context.Background())
	}
	if calls != 3 {
		t.Fatalf("sender called %d times", calls)
	}
	if len(logs) != 2 || !strings.Contains(logs[0], "mirror claude snapshot: network down") || !strings.Contains(logs[1], "mirror recovered for claude at ") {
		t.Fatalf("unexpected log calls: %v", logs)
	}
}

func TestLatestProvidersBreaksTimestampTiesDeterministically(t *testing.T) {
	now := time.Now()
	snapshots := []model.Snapshot{
		{Provider: "Claude", Session: model.Session{ID: "session-b"}, CapturedAt: now, Activity: model.Activity{UpdatedAt: now}},
		{Provider: "claude", Session: model.Session{ID: "session-a"}, CapturedAt: now, Activity: model.Activity{UpdatedAt: now.Add(time.Second)}},
	}
	selected := latestProviders(snapshots)
	if len(selected) != 1 || selected[0].Session.ID != "session-a" {
		t.Fatalf("selected snapshot = %+v", selected)
	}

	snapshots[0].Activity.UpdatedAt = snapshots[1].Activity.UpdatedAt
	selected = latestProviders(snapshots)
	if len(selected) != 1 || selected[0].Session.ID != "session-b" {
		t.Fatalf("session-ID tie break selected = %+v", selected)
	}
}

func TestSyncSendsValidSnapshotsAlongsideCorruptionAndLogsRecovery(t *testing.T) {
	store := testStore(t)
	saveSnapshot(t, store, "claude", "current", time.Now(), "working")
	brokenPath := filepath.Join(store.Dir(), "sessions", "broken.json")
	if err := os.WriteFile(brokenPath, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	calls := 0
	var logs []string
	relay, err := New(store, func(_ context.Context, snapshot model.Snapshot) error {
		calls++
		return nil
	}, func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := relay.Sync(context.Background()); err == nil {
		t.Fatal("Sync() accepted a corrupt session without reporting it")
	}
	if calls != 1 {
		t.Fatalf("valid snapshot sent %d times", calls)
	}
	if err := os.Remove(brokenPath); err != nil {
		t.Fatal(err)
	}
	if err := relay.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(logs) != 3 || !strings.Contains(logs[0], "load snapshots: ignored 1 invalid session snapshot") || !strings.HasPrefix(logs[1], "mirrored claude snapshot captured at ") || logs[2] != "snapshot store recovered" {
		t.Fatalf("unexpected recovery logs: %v", logs)
	}
}

func testStore(t *testing.T) *state.Store {
	t.Helper()
	store, err := state.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func saveSnapshot(t *testing.T, store *state.Store, provider, sessionID string, capturedAt time.Time, activity string) model.Snapshot {
	t.Helper()
	snapshot := model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion,
		Provider:      provider,
		Session:       model.Session{ID: sessionID},
		CapturedAt:    capturedAt,
		Activity:      model.Activity{State: activity, UpdatedAt: capturedAt},
	}
	if err := store.Save(snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

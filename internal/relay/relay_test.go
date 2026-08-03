package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

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
	}, zerolog.Nop())
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
	var buf bytes.Buffer
	relay, err := New(store, func(_ context.Context, _ model.Snapshot) error {
		calls++
		return nil
	}, zerolog.New(&buf))
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
	logs := parseLogs(t, &buf)
	if len(logs) != 1 || logs[0]["level"] != "info" || logs[0]["message"] != "mirrored snapshot" || logs[0]["provider"] != "claude" {
		t.Fatalf("successful refreshes logged repeatedly: %v", logs)
	}
}

func TestSyncRetriesFailureAndSuppressesRepeatedErrorLog(t *testing.T) {
	store := testStore(t)
	saveSnapshot(t, store, "claude", "current", time.Now(), "working")

	calls := 0
	var buf bytes.Buffer
	relay, err := New(store, func(_ context.Context, _ model.Snapshot) error {
		calls++
		if calls < 3 {
			return errors.New("network down")
		}
		return nil
	}, zerolog.New(&buf))
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		_ = relay.Sync(context.Background())
	}
	if calls != 3 {
		t.Fatalf("sender called %d times", calls)
	}
	logs := parseLogs(t, &buf)
	if len(logs) != 2 {
		t.Fatalf("unexpected log calls: %v", logs)
	}
	if logs[0]["level"] != "warn" || logs[0]["message"] != "mirror failed" || logs[0]["provider"] != "claude" || !strings.Contains(fmt.Sprint(logs[0]["error"]), "mirror claude snapshot: network down") {
		t.Fatalf("unexpected failure log: %v", logs[0])
	}
	if logs[1]["level"] != "info" || logs[1]["message"] != "mirror recovered" || logs[1]["provider"] != "claude" {
		t.Fatalf("unexpected recovery log: %v", logs[1])
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
	var buf bytes.Buffer
	relay, err := New(store, func(_ context.Context, _ model.Snapshot) error {
		calls++
		return nil
	}, zerolog.New(&buf))
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
	logs := parseLogs(t, &buf)
	if len(logs) != 3 {
		t.Fatalf("unexpected recovery logs: %v", logs)
	}
	if logs[0]["level"] != "warn" || logs[0]["message"] != "load snapshots" || !strings.Contains(fmt.Sprint(logs[0]["error"]), "ignored 1 invalid session snapshot") {
		t.Fatalf("unexpected load-error log: %v", logs[0])
	}
	if logs[1]["level"] != "info" || logs[1]["message"] != "mirrored snapshot" || logs[1]["provider"] != "claude" {
		t.Fatalf("unexpected mirrored log: %v", logs[1])
	}
	if logs[2]["level"] != "info" || logs[2]["message"] != "snapshot store recovered" {
		t.Fatalf("unexpected recovery log: %v", logs[2])
	}
}

// TestLogLoadErrorUsesDebugForTransientReadRace pins the log-noise fix: a
// reader landing mid atomic-rename (state.ErrTransientRead) is expected and
// self-resolving, so it must not log at the same visibility as real
// corruption.
func TestLogLoadErrorUsesDebugForTransientReadRace(t *testing.T) {
	var buf bytes.Buffer
	relay := &Relay{logger: zerolog.New(&buf)}
	relay.logLoadError(fmt.Errorf("read session: %w", state.ErrTransientRead))
	logs := parseLogs(t, &buf)
	if len(logs) != 1 || logs[0]["level"] != "debug" || logs[0]["message"] != "load snapshots" {
		t.Fatalf("transient read race should log at debug: %v", logs)
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

func parseLogs(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for raw := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if raw == "" {
			continue
		}
		var line map[string]any
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			t.Fatalf("parse log line %q: %v", raw, err)
		}
		lines = append(lines, line)
	}
	return lines
}

package usage

import (
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/state"
)

func TestRunMergesRateLimitsWithoutTouchingOtherFields(t *testing.T) {
	store, err := state.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	seeded := model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion,
		CapturedAt:    time.Now(),
		Provider:      "claude",
		Session:       model.Session{ID: "abc", Name: "demo"},
		Model:         model.Model{DisplayName: "Sonnet 5"},
		Activity:      model.Activity{State: model.ActivityWorking, UpdatedAt: time.Now()},
	}
	if err := store.Save(seeded); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	snapshot, err := Run(store, "abc", 63, 42, 5*time.Hour, 168*time.Hour, now)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if *snapshot.RateLimits.FiveHour.UsedPercentage != 63 {
		t.Fatalf("FiveHour.UsedPercentage = %v", *snapshot.RateLimits.FiveHour.UsedPercentage)
	}
	if *snapshot.RateLimits.SevenDay.UsedPercentage != 42 {
		t.Fatalf("SevenDay.UsedPercentage = %v", *snapshot.RateLimits.SevenDay.UsedPercentage)
	}
	wantFiveReset := now.Add(5 * time.Hour).Unix()
	if *snapshot.RateLimits.FiveHour.ResetsAt != wantFiveReset {
		t.Fatalf("FiveHour.ResetsAt = %v, want %v", *snapshot.RateLimits.FiveHour.ResetsAt, wantFiveReset)
	}
	wantSevenReset := now.Add(168 * time.Hour).Unix()
	if *snapshot.RateLimits.SevenDay.ResetsAt != wantSevenReset {
		t.Fatalf("SevenDay.ResetsAt = %v, want %v", *snapshot.RateLimits.SevenDay.ResetsAt, wantSevenReset)
	}

	if snapshot.Model.DisplayName != "Sonnet 5" || snapshot.Session.Name != "demo" {
		t.Fatalf("Run() clobbered unrelated fields: %+v", snapshot)
	}
	if snapshot.Activity.State != model.ActivityWorking {
		t.Fatalf("Run() clobbered activity: %+v", snapshot.Activity)
	}
}

func TestRunDefaultsToTheMostRecentlyUpdatedSession(t *testing.T) {
	store, err := state.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion, CapturedAt: time.Now(), Session: model.Session{ID: "latest-one"},
	}); err != nil {
		t.Fatal(err)
	}

	snapshot, err := Run(store, "", 10, 20, time.Hour, 24*time.Hour, time.Now())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if snapshot.Session.ID != "latest-one" {
		t.Fatalf("Run() updated session %q, want %q", snapshot.Session.ID, "latest-one")
	}
}

func TestRunClampsOutOfRangePercentages(t *testing.T) {
	store, err := state.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(model.Snapshot{SchemaVersion: model.CurrentSchemaVersion, CapturedAt: time.Now(), Session: model.Session{ID: "abc"}}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Run(store, "abc", 150, -5, time.Hour, time.Hour, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if *snapshot.RateLimits.FiveHour.UsedPercentage != 100 {
		t.Fatalf("FiveHour.UsedPercentage = %v, want clamped to 100", *snapshot.RateLimits.FiveHour.UsedPercentage)
	}
	if *snapshot.RateLimits.SevenDay.UsedPercentage != 0 {
		t.Fatalf("SevenDay.UsedPercentage = %v, want clamped to 0", *snapshot.RateLimits.SevenDay.UsedPercentage)
	}
}

func TestRunReturnsErrorWhenNoSnapshotExists(t *testing.T) {
	store, err := state.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(store, "missing", 1, 1, time.Hour, time.Hour, time.Now()); err == nil {
		t.Fatal("expected an error when the target session has no existing snapshot")
	}
}

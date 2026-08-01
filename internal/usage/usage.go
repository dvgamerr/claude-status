// Package usage merges manually-supplied 5h/7d rate-limit percentages into
// an already-stored snapshot. It exists for interfaces where Claude Code's
// statusLine command is never invoked (this project has only ever seen
// that happen from a real CLI terminal, never from the VS Code extension
// chat panel) — a manual stand-in for the numbers statusLine would
// otherwise report, read off the Account & Usage panel by hand.
package usage

import (
	"fmt"
	"math"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/state"
)

// Run loads the target snapshot (by session ID, or the most recently
// updated session if sessionID is empty), overwrites only its RateLimits,
// and saves it back — model, context, session, and activity are left
// exactly as they were.
func Run(store *state.Store, sessionID string, fiveHourPercent, sevenDayPercent float64, fiveHourReset, sevenDayReset time.Duration, now time.Time) (model.Snapshot, error) {
	snapshot, err := load(store, sessionID)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("load existing snapshot: %w", err)
	}

	five := clampPercent(fiveHourPercent)
	seven := clampPercent(sevenDayPercent)
	fiveResetAt := now.Add(fiveHourReset).Unix()
	sevenResetAt := now.Add(sevenDayReset).Unix()
	snapshot.RateLimits.FiveHour = model.RateWindow{UsedPercentage: &five, ResetsAt: &fiveResetAt}
	snapshot.RateLimits.SevenDay = model.RateWindow{UsedPercentage: &seven, ResetsAt: &sevenResetAt}

	if err := store.Save(snapshot); err != nil {
		return model.Snapshot{}, fmt.Errorf("save snapshot: %w", err)
	}
	return snapshot, nil
}

func load(store *state.Store, sessionID string) (model.Snapshot, error) {
	if sessionID == "" {
		return store.LoadLatest()
	}
	return store.LoadSession(sessionID)
}

func clampPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return min(100, max(0, value))
}

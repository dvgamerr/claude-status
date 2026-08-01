// Package activity turns a Claude Code hook event into an update of the
// dashboard's working/idle/waiting-approval indicator, independent of the
// statusLine refresh cycle.
package activity

import (
	"io"
	"os"
	"time"

	"github.com/dvgamerr/claude-status/internal/claude"
	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/state"
)

// Run reads one Claude Code hook payload from input and, if it maps to a
// meaningful activity state, merges that state into the session's stored
// snapshot. matched is false when the hook event is one the dashboard
// ignores (for example an idle-nudge Notification); callers should treat
// that as a no-op, not an error.
func Run(input io.Reader, store *state.Store, now time.Time) (snapshot model.Snapshot, matched bool, err error) {
	hook, err := claude.DecodeHook(input)
	if err != nil {
		return model.Snapshot{}, false, err
	}
	newState, ok := claude.ActivityForHook(hook)
	if !ok {
		return model.Snapshot{}, false, nil
	}
	sessionID := claude.CleanSessionID(hook.SessionID)

	snapshot, err = store.LoadSession(sessionID)
	if err != nil {
		if !os.IsNotExist(err) {
			return model.Snapshot{}, false, err
		}
		snapshot = model.Snapshot{
			SchemaVersion: model.CurrentSchemaVersion,
			CapturedAt:    now,
			Provider:      "claude",
			Session:       model.Session{ID: sessionID},
		}
	}
	snapshot.Activity = model.Activity{State: newState, UpdatedAt: now}
	if err := store.Save(snapshot); err != nil {
		return model.Snapshot{}, false, err
	}
	return snapshot, true, nil
}

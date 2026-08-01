package ingest

import (
	"fmt"
	"io"
	"time"

	"github.com/dvgamerr/claude-status/internal/claude"
	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/state"
)

func Run(input io.Reader, output io.Writer, store *state.Store, now time.Time) (model.Snapshot, error) {
	payload, err := claude.Decode(input)
	if err != nil {
		return model.Snapshot{}, err
	}
	snapshot := claude.ToSnapshot(payload, now)
	if err := store.Save(snapshot); err != nil {
		return model.Snapshot{}, err
	}
	if _, err := fmt.Fprintln(output, claude.StatusLine(snapshot)); err != nil {
		return model.Snapshot{}, fmt.Errorf("write status line: %w", err)
	}
	return snapshot, nil
}

// Package relay mirrors locally persisted snapshots from short-lived source
// commands to a remote dashboard. Keeping transport in a long-lived process
// means Claude statusLine and hook commands never have to wait for SSH.
package relay

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/state"
)

type Sender func(context.Context, model.Snapshot) error
type LogFunc func(string, ...any)

type Relay struct {
	store         *state.Store
	send          Sender
	logf          LogFunc
	sent          map[string][sha256.Size]byte
	failures      map[string]string
	lastLoadError string
}

func New(store *state.Store, send Sender, logf LogFunc) (*Relay, error) {
	if store == nil {
		return nil, errors.New("snapshot store is nil")
	}
	if send == nil {
		return nil, errors.New("snapshot sender is nil")
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Relay{
		store:    store,
		send:     send,
		logf:     logf,
		sent:     make(map[string][sha256.Size]byte),
		failures: make(map[string]string),
	}, nil
}

// Sync sends the newest locally persisted snapshot for every provider whose
// sanitized content has changed since its last successful delivery. Failed
// deliveries stay pending and are retried by the next call.
func (r *Relay) Sync(ctx context.Context) error {
	snapshots, loadErr := r.store.LoadAll()
	r.logLoadError(loadErr)

	selected := latestProviders(snapshots)
	sort.SliceStable(selected, func(i, j int) bool {
		if selected[i].CapturedAt.Equal(selected[j].CapturedAt) {
			return providerKey(selected[i]) < providerKey(selected[j])
		}
		return selected[i].CapturedAt.Before(selected[j].CapturedAt)
	})

	var syncErrors []error
	if loadErr != nil {
		syncErrors = append(syncErrors, loadErr)
	}
	for _, snapshot := range selected {
		key := providerKey(snapshot)
		fingerprint, err := snapshotFingerprint(snapshot)
		if err != nil {
			syncErrors = append(syncErrors, err)
			continue
		}
		if delivered, ok := r.sent[key]; ok && delivered == fingerprint {
			continue
		}

		if err := r.send(ctx, snapshot); err != nil {
			wrapped := fmt.Errorf("mirror %s snapshot: %w", key, err)
			r.logFailure(key, wrapped)
			syncErrors = append(syncErrors, wrapped)
			continue
		}
		_, alreadyDelivered := r.sent[key]
		r.sent[key] = fingerprint
		if _, recovering := r.failures[key]; recovering {
			r.logf("mirror recovered for %s at %s", key, snapshot.CapturedAt.Format(time.RFC3339Nano))
		} else if !alreadyDelivered {
			r.logf("mirrored %s snapshot captured at %s", key, snapshot.CapturedAt.Format(time.RFC3339Nano))
		}
		delete(r.failures, key)
	}
	return errors.Join(syncErrors...)
}

func (r *Relay) logLoadError(err error) {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	if detail == r.lastLoadError {
		return
	}
	r.lastLoadError = detail
	if detail != "" {
		r.logf("load snapshots: %s", detail)
	} else {
		r.logf("snapshot store recovered")
	}
}

func (r *Relay) logFailure(key string, err error) {
	detail := err.Error()
	if r.failures[key] == detail {
		return
	}
	r.failures[key] = detail
	r.logf("%s", detail)
}

func latestProviders(snapshots []model.Snapshot) []model.Snapshot {
	latest := make(map[string]model.Snapshot)
	for _, snapshot := range snapshots {
		key := providerKey(snapshot)
		current, ok := latest[key]
		if !ok || snapshotIsNewer(snapshot, current) {
			latest[key] = snapshot
		}
	}
	selected := make([]model.Snapshot, 0, len(latest))
	for _, snapshot := range latest {
		selected = append(selected, snapshot)
	}
	return selected
}

func snapshotIsNewer(candidate, current model.Snapshot) bool {
	if !candidate.CapturedAt.Equal(current.CapturedAt) {
		return candidate.CapturedAt.After(current.CapturedAt)
	}
	if !candidate.Activity.UpdatedAt.Equal(current.Activity.UpdatedAt) {
		return candidate.Activity.UpdatedAt.After(current.Activity.UpdatedAt)
	}
	return candidate.Session.ID > current.Session.ID
}

func providerKey(snapshot model.Snapshot) string {
	provider := strings.ToLower(strings.TrimSpace(snapshot.Provider))
	if provider != "" {
		return provider
	}
	return "session:" + snapshot.Session.ID
}

func snapshotFingerprint(snapshot model.Snapshot) ([sha256.Size]byte, error) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("encode %s snapshot: %w", providerKey(snapshot), err)
	}
	return sha256.Sum256(data), nil
}

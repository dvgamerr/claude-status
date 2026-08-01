package pixelui

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
)

type SnapshotLoader interface {
	LoadAll() ([]model.Snapshot, error)
}

type MetricsReader interface {
	Read() systeminfo.Stats
}

type Screen interface {
	Present(image.Image) error
}

type RunConfig struct {
	RefreshInterval time.Duration
	StaleAfter      time.Duration
}

func Run(ctx context.Context, loader SnapshotLoader, metrics MetricsReader, screen Screen, renderer *Renderer, config RunConfig) error {
	if config.RefreshInterval <= 0 {
		config.RefreshInterval = time.Second
	}
	if config.StaleAfter <= 0 {
		config.StaleAfter = 15 * time.Second
	}
	if err := renderFrame(loader, metrics, screen, renderer, config, time.Now()); err != nil {
		return err
	}
	ticker := time.NewTicker(config.RefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			if err := renderFrame(loader, metrics, screen, renderer, config, now); err != nil {
				return err
			}
		}
	}
}

func renderFrame(loader SnapshotLoader, metrics MetricsReader, screen Screen, renderer *Renderer, config RunConfig, now time.Time) error {
	snapshots, loadErr := loader.LoadAll()
	claude, codex := LatestProviders(snapshots)
	frame := renderer.Render(View{
		Claude:       claude,
		Codex:        codex,
		Stats:        metrics.Read(),
		Now:          now,
		StaleAfter:   config.StaleAfter,
		SessionCount: len(snapshots),
		LoadError:    loadErr,
	})
	if err := screen.Present(frame); err != nil {
		return fmt.Errorf("present pixel dashboard: %w", err)
	}
	return nil
}

// LatestProviders keeps Claude and Codex independent so a newer Codex event
// cannot displace the Claude-first dashboard.
func LatestProviders(snapshots []model.Snapshot) (*model.Snapshot, *model.Snapshot) {
	var claude, codex *model.Snapshot
	for index := range snapshots {
		snapshot := snapshots[index]
		switch strings.ToLower(strings.TrimSpace(snapshot.Provider)) {
		case "claude":
			if claude == nil || snapshot.CapturedAt.After(claude.CapturedAt) {
				copy := snapshot
				claude = &copy
			}
		case "codex":
			if codex == nil || snapshot.CapturedAt.After(codex.CapturedAt) {
				copy := snapshot
				codex = &copy
			}
		}
	}
	return claude, codex
}

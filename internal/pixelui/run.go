package pixelui

import (
	"context"
	"fmt"
	"image"
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
	var latest *model.Snapshot
	if len(snapshots) > 0 {
		copy := snapshots[0]
		latest = &copy
	}
	frame := renderer.Render(View{
		Snapshot:     latest,
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

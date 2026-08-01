package pixelui

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
	"github.com/dvgamerr/claude-status/internal/touch"
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

// Run renders frames on config.RefreshInterval. touches may be nil (no
// touch input available, e.g. the device failed to open) — a nil channel
// always blocks in a select with a default case, so touch feedback is
// simply absent rather than a special case to handle.
func Run(ctx context.Context, loader SnapshotLoader, metrics MetricsReader, screen Screen, renderer *Renderer, config RunConfig, touches <-chan touch.Point) error {
	if config.RefreshInterval <= 0 {
		config.RefreshInterval = time.Second
	}
	if config.StaleAfter <= 0 {
		config.StaleAfter = 15 * time.Second
	}
	var active []touch.Point
	active = drainTouches(touches, active, time.Now())
	if err := renderFrame(loader, metrics, screen, renderer, config, time.Now(), active); err != nil {
		return err
	}
	ticker := time.NewTicker(config.RefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			active = drainTouches(touches, active, now)
			if err := renderFrame(loader, metrics, screen, renderer, config, now, active); err != nil {
				return err
			}
		}
	}
}

// drainTouches collects any newly arrived touch points without blocking,
// then drops points too old for the ripple effect to still be visible.
func drainTouches(touches <-chan touch.Point, active []touch.Point, now time.Time) []touch.Point {
	for {
		select {
		case point := <-touches:
			active = append(active, point)
		default:
			kept := active[:0]
			for _, point := range active {
				if now.Sub(point.At) <= touchRippleLifetime {
					kept = append(kept, point)
				}
			}
			return kept
		}
	}
}

func renderFrame(loader SnapshotLoader, metrics MetricsReader, screen Screen, renderer *Renderer, config RunConfig, now time.Time, touches []touch.Point) error {
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
		Touches:      touches,
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

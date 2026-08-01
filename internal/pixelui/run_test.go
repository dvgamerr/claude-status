package pixelui

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
	"github.com/dvgamerr/claude-status/internal/touch"
)

type testLoader struct {
	snapshots []model.Snapshot
	err       error
}

func (loader testLoader) LoadAll() ([]model.Snapshot, error) { return loader.snapshots, loader.err }

type testMetrics struct{ stats systeminfo.Stats }

func (metrics testMetrics) Read() systeminfo.Stats { return metrics.stats }

type testScreen struct {
	frames int
	err    error
}

func (screen *testScreen) Present(frame image.Image) error {
	if frame.Bounds() != image.Rect(0, 0, Width, Height) {
		return errors.New("wrong frame size")
	}
	screen.frames++
	return screen.err
}

func TestRunRendersImmediatelyAndStopsWithContext(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	screen := &testScreen{}
	err = Run(ctx, testLoader{}, testMetrics{}, screen, renderer, RunConfig{RefreshInterval: time.Second}, nil)
	if err != nil || screen.frames != 1 {
		t.Fatalf("Run() error=%v frames=%d", err, screen.frames)
	}
}

func TestRenderFramePropagatesPresentError(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	screen := &testScreen{err: errors.New("screen failed")}
	err = renderFrame(testLoader{err: errors.New("partial state")}, testMetrics{}, screen, renderer, RunConfig{}, time.Now(), nil)
	if err == nil || !errors.Is(err, screen.err) {
		t.Fatalf("renderFrame() error = %v", err)
	}
}

func TestDrainTouchesCollectsAndExpiresPoints(t *testing.T) {
	now := time.Now()
	channel := make(chan touch.Point, 2)
	channel <- touch.Point{X: 1, Y: 2, At: now}
	channel <- touch.Point{X: 3, Y: 4, At: now}

	active := drainTouches(channel, nil, now)
	if len(active) != 2 {
		t.Fatalf("drainTouches() collected %d points, want 2", len(active))
	}

	active = drainTouches(channel, active, now.Add(touchRippleLifetime+time.Millisecond))
	if len(active) != 0 {
		t.Fatalf("drainTouches() kept %d expired points, want 0", len(active))
	}
}

func TestLatestProvidersKeepsClaudePrimaryWhenCodexIsNewer(t *testing.T) {
	now := time.Now()
	snapshots := []model.Snapshot{
		{Provider: "codex", CapturedAt: now, Session: model.Session{ID: "codex-new"}},
		{Provider: "claude", CapturedAt: now.Add(-time.Hour), Session: model.Session{ID: "claude-old"}},
		{Provider: "claude", CapturedAt: now.Add(-time.Minute), Session: model.Session{ID: "claude-current"}},
	}
	claude, codex := LatestProviders(snapshots)
	if claude == nil || claude.Session.ID != "claude-current" {
		t.Fatalf("Claude selection = %+v", claude)
	}
	if codex == nil || codex.Session.ID != "codex-new" {
		t.Fatalf("Codex selection = %+v", codex)
	}
}

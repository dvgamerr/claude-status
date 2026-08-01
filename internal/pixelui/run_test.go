package pixelui

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
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
	err = Run(ctx, testLoader{}, testMetrics{}, screen, renderer, RunConfig{RefreshInterval: time.Second})
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
	err = renderFrame(testLoader{err: errors.New("partial state")}, testMetrics{}, screen, renderer, RunConfig{}, time.Now())
	if err == nil || !errors.Is(err, screen.err) {
		t.Fatalf("renderFrame() error = %v", err)
	}
}

package pixelui

import (
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
)

func TestRenderDashboardAndWaitingFrames(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	contextPct, fivePct, sevenPct := 72.0, 51.0, 91.0
	window, input, output := int64(200000), int64(140000), int64(4000)
	resetFive, resetSeven := now.Add(90*time.Minute).Unix(), now.Add(72*time.Hour).Unix()
	cpu, temperature, load := 18.0, 52.0, 0.42
	usedMemory, totalMemory := uint64(1<<30), uint64(4<<30)
	snapshot := model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion,
		CapturedAt:    now.Add(-4 * time.Second),
		Provider:      "claude",
		ClientVersion: "2.1.220",
		Session:       model.Session{ID: "session-123", Name: "Pi dashboard demo"},
		Model:         model.Model{ID: "claude-opus", DisplayName: "Opus"},
		Context:       model.Context{UsedPercentage: &contextPct, WindowSize: &window, TotalInputTokens: &input, TotalOutputTokens: &output},
		RateLimits: model.RateLimits{
			FiveHour: model.RateWindow{UsedPercentage: &fivePct, ResetsAt: &resetFive},
			SevenDay: model.RateWindow{UsedPercentage: &sevenPct, ResetsAt: &resetSeven},
		},
		Effort: "high",
	}
	frame := renderer.Render(View{
		Snapshot: &snapshot,
		Stats: systeminfo.Stats{
			CPUPercent: &cpu, TemperatureC: &temperature, Load1: &load,
			MemoryUsedBytes: &usedMemory, MemoryTotalBytes: &totalMemory,
		},
		Now: now, StaleAfter: 15 * time.Second, SessionCount: 2,
	})
	if frame.Bounds() != image.Rect(0, 0, Width, Height) {
		t.Fatalf("frame bounds = %v", frame.Bounds())
	}
	for _, point := range []image.Point{{30, 25}, {30, 80}, {30, 250}, {420, 250}, {30, 370}} {
		if got := rgba(frame.At(point.X, point.Y)); got == backgroundTop {
			t.Fatalf("expected card/content at %v, got background %v", point, got)
		}
	}
	waiting := renderer.Render(View{Now: now, LoadError: errors.New("waiting")})
	if waiting.Bounds() != frame.Bounds() || rgba(waiting.At(400, 171)) == backgroundTop {
		t.Fatalf("waiting frame was not rendered")
	}
}

func TestRenderCodexUnlimitedAndStale(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	unlimited := true
	pct := 42.5
	snapshot := model.Snapshot{
		SchemaVersion:   model.CurrentSchemaVersion,
		CapturedAt:      now.Add(-time.Minute),
		Provider:        "codex",
		Session:         model.Session{ID: strings.Repeat("a", 40)},
		Context:         model.Context{UsedPercentage: &pct},
		RateLimits:      model.RateLimits{Plan: "enterprise_cbp_usage_based", Unlimited: &unlimited},
		ThinkingEnabled: boolPointer(true),
	}
	frame := renderer.Render(View{Snapshot: &snapshot, Now: now, StaleAfter: time.Second, SessionCount: 1})
	if frame.Bounds().Dx() != Width || frame.Bounds().Dy() != Height {
		t.Fatalf("unexpected frame size %v", frame.Bounds())
	}
	if providerName(snapshot.Provider) != "CODEX" || providerAccent(snapshot.Provider) != green {
		t.Fatal("Codex provider theme was not selected")
	}
}

func TestFormattingHelpers(t *testing.T) {
	thousand, million := int64(1200), int64(2_000_000)
	if tokenLabel(&thousand) != "1.2K" || tokenLabel(&million) != "2M" || tokenLabel(nil) != "--" {
		t.Fatalf("token labels are incorrect")
	}
	if got := durationLabel(26*time.Hour + 3*time.Minute); got != "1d 2h" {
		t.Fatalf("durationLabel() = %q", got)
	}
	if got := thresholdColor(95, blue); got != red {
		t.Fatalf("thresholdColor(95) = %v", got)
	}
	if got := thresholdColor(75, blue); got != yellow {
		t.Fatalf("thresholdColor(75) = %v", got)
	}
	if resetLabel(nil, time.Now()) != "reset unavailable" {
		t.Fatal("missing reset label is incorrect")
	}
	if ageText(0) != "just now" || ageText(2*time.Minute) != "2m ago" {
		t.Fatal("age labels are incorrect")
	}
}

func rgba(value color.Color) color.RGBA {
	red, green, blue, alpha := value.RGBA()
	return color.RGBA{R: uint8(red >> 8), G: uint8(green >> 8), B: uint8(blue >> 8), A: uint8(alpha >> 8)}
}

func boolPointer(value bool) *bool { return &value }

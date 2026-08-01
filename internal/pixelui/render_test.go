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
		Claude: &snapshot,
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

func TestRenderClaudePrimaryWithCodexCompactAndAnimatedMark(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	claudePct, codexPct := 72.0, 42.5
	claude := model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion,
		CapturedAt:    now.Add(-2 * time.Second),
		Provider:      "claude",
		Session:       model.Session{ID: "claude-session", Name: "Primary Claude work"},
		Model:         model.Model{DisplayName: "Opus"},
		Context:       model.Context{UsedPercentage: &claudePct},
	}
	codex := model.Snapshot{
		SchemaVersion:   model.CurrentSchemaVersion,
		CapturedAt:      now.Add(-time.Minute),
		Provider:        "codex",
		Session:         model.Session{ID: strings.Repeat("a", 40)},
		Model:           model.Model{ID: "gpt-5.6-sol"},
		Context:         model.Context{UsedPercentage: &codexPct},
		ThinkingEnabled: boolPointer(true),
	}
	frame := renderer.Render(View{Claude: &claude, Codex: &codex, Now: now, StaleAfter: 15 * time.Second, SessionCount: 2})
	if frame.Bounds().Dx() != Width || frame.Bounds().Dy() != Height {
		t.Fatalf("unexpected frame size %v", frame.Bounds())
	}
	if rgba(frame.At(550, 365)) == backgroundTop {
		t.Fatal("Codex compact card was not rendered")
	}
	later := renderer.Render(View{Claude: &claude, Codex: &codex, Now: now.Add(250 * time.Millisecond), StaleAfter: 15 * time.Second, SessionCount: 2})
	if sameRegion(frame, later, image.Rect(12, 6, 68, 62)) {
		t.Fatal("Claude mark did not animate between frames")
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
	if got := thresholdColor(95, claudeOrange); got != red {
		t.Fatalf("thresholdColor(95) = %v", got)
	}
	if got := thresholdColor(75, claudeOrange); got != yellow {
		t.Fatalf("thresholdColor(75) = %v", got)
	}
	if resetLabel(nil, time.Now()) != "reset unavailable" {
		t.Fatal("missing reset label is incorrect")
	}
	if ageText(0) != "just now" || ageText(2*time.Minute) != "2m ago" {
		t.Fatal("age labels are incorrect")
	}
}

func TestQuotaCardHandlesUnlimitedAndUnavailableValues(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, Width, Height))
	unlimited := true
	renderer.quotaCard(canvas, image.Rect(18, 20, 393, 128), "5 HOUR", model.RateWindow{}, model.RateLimits{Unlimited: &unlimited, Plan: "enterprise"}, claudeOrange, time.Now())
	renderer.quotaCard(canvas, image.Rect(407, 20, 782, 128), "7 DAY", model.RateWindow{}, model.RateLimits{}, claudePeach, time.Now())
	if rgba(canvas.At(30, 30)) == (color.RGBA{}) || rgba(canvas.At(420, 30)) == (color.RGBA{}) {
		t.Fatal("quota cards were not painted")
	}
}

func rgba(value color.Color) color.RGBA {
	red, green, blue, alpha := value.RGBA()
	return color.RGBA{R: uint8(red >> 8), G: uint8(green >> 8), B: uint8(blue >> 8), A: uint8(alpha >> 8)}
}

func sameRegion(left, right *image.RGBA, bounds image.Rectangle) bool {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if rgba(left.At(x, y)) != rgba(right.At(x, y)) {
				return false
			}
		}
	}
	return true
}

func boolPointer(value bool) *bool { return &value }

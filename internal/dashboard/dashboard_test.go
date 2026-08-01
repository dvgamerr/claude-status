package dashboard

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	statusmodel "github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
)

type fakeLoader struct {
	snapshots []statusmodel.Snapshot
	err       error
}

func (f fakeLoader) LoadAll() ([]statusmodel.Snapshot, error) { return f.snapshots, f.err }

type fakeMetrics struct{ stats systeminfo.Stats }

func (f fakeMetrics) Read() systeminfo.Stats { return f.stats }

func TestDashboardViewAndSessionSelection(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.Local)
	pct5, pct7, contextPct := 51.0, 34.0, 72.0
	window := int64(200000)
	input := int64(140000)
	output := int64(4000)
	duration := int64(6_138_000)
	cost := 1.28
	added, removed := int64(186), int64(42)
	reset := now.Add(2 * time.Hour).Unix()
	snapshots := []statusmodel.Snapshot{
		{
			SchemaVersion: modelSchema(), CapturedAt: now.Add(-4 * time.Second),
			Session: statusmodel.Session{ID: "abc1234567890123", Name: "primary"},
			Model:   statusmodel.Model{DisplayName: "Opus"},
			RateLimits: statusmodel.RateLimits{
				FiveHour: statusmodel.RateWindow{UsedPercentage: &pct5, ResetsAt: &reset},
				SevenDay: statusmodel.RateWindow{UsedPercentage: &pct7, ResetsAt: &reset},
			},
			Context: statusmodel.Context{UsedPercentage: &contextPct, WindowSize: &window, TotalInputTokens: &input, TotalOutputTokens: &output},
			Cost:    statusmodel.Cost{TotalDurationMS: &duration, TotalCostUSD: &cost, TotalLinesAdded: &added, TotalLinesRemoved: &removed},
		},
		{SchemaVersion: modelSchema(), CapturedAt: now.Add(-time.Minute), Session: statusmodel.Session{ID: "second"}, Model: statusmodel.Model{DisplayName: "Sonnet"}},
	}
	cpu, temp := 18.0, 52.0
	used, total := uint64(1<<30), uint64(4<<30)
	uptime := 102*time.Minute + 18*time.Second
	m := NewModel(fakeLoader{snapshots: snapshots}, fakeMetrics{systeminfo.Stats{CPUPercent: &cpu, TemperatureC: &temp, MemoryUsedBytes: &used, MemoryTotalBytes: &total, Uptime: &uptime}}, Config{})
	m.now = func() time.Time { return now }
	updated, _ := m.Update(dataMsg{snapshots: snapshots, stats: m.metrics.Read()})
	m = updated.(Model)
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 66, Height: 20})
	m = updated.(Model)

	view := m.View()
	for _, want := range []string{"CLAUDE STATUS", "LIVE", "OPUS", "5-HOUR LIMIT", "7-DAY LIMIT", "51%", "34%", "CONTEXT", "72%", "144k / 200k", "↑ 140k", "↓ 4k", "$1.28", "+186  -42", "CPU 18%", "Temp 52°C", "Up 1h42m", "primary"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() does not contain %q:\n%s", want, view)
		}
	}
	if got := lipgloss.Width(view); got != 66 {
		t.Fatalf("Pi dashboard width = %d, want 66:\n%s", got, view)
	}
	if got := lipgloss.Height(view); got != 20 {
		t.Fatalf("Pi dashboard height = %d, want 20:\n%s", got, view)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(Model)
	if !strings.Contains(m.View(), "SESSIONS") || !strings.Contains(m.View(), "Sonnet") {
		t.Fatalf("sessions view is incomplete:\n%s", m.View())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.selectedID != "second" || m.showSessions {
		t.Fatalf("session selection failed: selected=%q show=%v", m.selectedID, m.showSessions)
	}
	if !m.selectionPinned {
		t.Fatal("manual session selection did not pin the session")
	}
}

func TestDashboardAutoFollowsNewestSnapshot(t *testing.T) {
	now := time.Now()
	older := statusmodel.Snapshot{SchemaVersion: modelSchema(), CapturedAt: now.Add(-time.Minute), Session: statusmodel.Session{ID: "older"}}
	newer := statusmodel.Snapshot{SchemaVersion: modelSchema(), CapturedAt: now, Session: statusmodel.Session{ID: "newer"}}
	m := NewModel(fakeLoader{}, fakeMetrics{}, Config{})
	updated, _ := m.Update(dataMsg{snapshots: []statusmodel.Snapshot{older}})
	m = updated.(Model)
	if m.selectedID != "older" {
		t.Fatalf("initial auto selection = %q", m.selectedID)
	}
	updated, _ = m.Update(dataMsg{snapshots: []statusmodel.Snapshot{newer, older}})
	m = updated.(Model)
	if m.selectedID != "newer" {
		t.Fatalf("auto selection did not follow newest snapshot: %q", m.selectedID)
	}

	m.selectionPinned = true
	m.selectedID = "older"
	updated, _ = m.Update(dataMsg{snapshots: []statusmodel.Snapshot{newer, older}})
	m = updated.(Model)
	if m.selectedID != "older" {
		t.Fatalf("pinned selection moved to %q", m.selectedID)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if m.selectionPinned || m.selectedID != "newer" {
		t.Fatalf("auto key did not resume newest selection: pinned=%v selected=%q", m.selectionPinned, m.selectedID)
	}
}

func TestDashboardMarksOldSnapshotStale(t *testing.T) {
	now := time.Now()
	snapshot := statusmodel.Snapshot{SchemaVersion: modelSchema(), CapturedAt: now.Add(-time.Minute), Session: statusmodel.Session{ID: "old"}}
	m := NewModel(fakeLoader{}, fakeMetrics{}, Config{StaleAfter: 15 * time.Second})
	m.now = func() time.Time { return now }
	updated, _ := m.Update(dataMsg{snapshots: []statusmodel.Snapshot{snapshot}})
	m = updated.(Model)
	if !strings.Contains(m.View(), "STALE") {
		t.Fatalf("old snapshot was not marked stale:\n%s", m.View())
	}
}

func TestDashboardLabelsCodexSnapshot(t *testing.T) {
	now := time.Now()
	snapshot := statusmodel.Snapshot{
		SchemaVersion: modelSchema(),
		CapturedAt:    now,
		Provider:      "codex",
		ClientVersion: "0.146.0",
		Session:       statusmodel.Session{ID: "codex-thread"},
		Model:         statusmodel.Model{ID: "gpt-5.6-sol"},
	}
	m := NewModel(fakeLoader{}, fakeMetrics{}, Config{})
	m.now = func() time.Time { return now }
	updated, _ := m.Update(dataMsg{snapshots: []statusmodel.Snapshot{snapshot}})
	m = updated.(Model)
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 66, Height: 20})
	m = updated.(Model)
	view := m.View()
	for _, want := range []string{"CODEX STATUS", "GPT-5.6-SOL", "CODEX 0.146.0"} {
		if !strings.Contains(view, want) {
			t.Fatalf("Codex dashboard does not contain %q:\n%s", want, view)
		}
	}
}

func TestDashboardDisplaysPartialLoadWarning(t *testing.T) {
	now := time.Now()
	snapshot := statusmodel.Snapshot{SchemaVersion: modelSchema(), CapturedAt: now, Session: statusmodel.Session{ID: "valid"}}
	m := NewModel(fakeLoader{}, fakeMetrics{}, Config{})
	m.now = func() time.Time { return now }
	updated, _ := m.Update(dataMsg{snapshots: []statusmodel.Snapshot{snapshot}, err: errors.New("ignored 1 invalid session snapshot")})
	m = updated.(Model)
	view := m.View()
	if !strings.Contains(view, "ignored 1 invalid session snapshot") || !strings.Contains(view, "valid") {
		t.Fatalf("partial load warning or valid session missing:\n%s", view)
	}
}

func TestDashboardFitsNarrowTerminal(t *testing.T) {
	now := time.Now()
	contextPct, fivePct, sevenPct := 18.0, 27.0, 44.0
	input, cacheCreate, cacheRead, output := int64(500), int64(1000), int64(1500), int64(750)
	snapshot := statusmodel.Snapshot{
		SchemaVersion: modelSchema(),
		CapturedAt:    now,
		Session:       statusmodel.Session{ID: "narrow"},
		Context: statusmodel.Context{
			UsedPercentage: &contextPct,
			CurrentUsage: statusmodel.TokenUsage{
				InputTokens:              &input,
				CacheCreationInputTokens: &cacheCreate,
				CacheReadInputTokens:     &cacheRead,
				OutputTokens:             &output,
			},
		},
		RateLimits: statusmodel.RateLimits{
			FiveHour: statusmodel.RateWindow{UsedPercentage: &fivePct},
			SevenDay: statusmodel.RateWindow{UsedPercentage: &sevenPct},
		},
	}
	m := NewModel(fakeLoader{}, fakeMetrics{}, Config{})
	m.now = func() time.Time { return now }
	updated, _ := m.Update(dataMsg{snapshots: []statusmodel.Snapshot{snapshot}})
	m = updated.(Model)
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = updated.(Model)
	view := m.View()
	if got := lipgloss.Width(view); got != 40 {
		t.Fatalf("dashboard width = %d, want 40:\n%s", got, view)
	}
	if got := lipgloss.Height(view); got != 20 {
		t.Fatalf("dashboard height = %d, want 20:\n%s", got, view)
	}
	for _, want := range []string{"CLAUDE STATUS", "CTX", "18%", "5H", "27%", "7D", "44%", "↑ 3k input", "↓ 750 output"} {
		if !strings.Contains(view, want) {
			t.Fatalf("compact dashboard does not contain %q:\n%s", want, view)
		}
	}
}

func TestSessionPickerPaginatesWithinPiDisplay(t *testing.T) {
	now := time.Now()
	snapshots := make([]statusmodel.Snapshot, 30)
	for i := range snapshots {
		snapshots[i] = statusmodel.Snapshot{
			SchemaVersion: modelSchema(),
			CapturedAt:    now.Add(-time.Duration(i) * time.Minute),
			Session:       statusmodel.Session{ID: fmt.Sprintf("session-%02d", i)},
		}
	}
	m := NewModel(fakeLoader{}, fakeMetrics{}, Config{})
	m.now = func() time.Time { return now }
	updated, _ := m.Update(dataMsg{snapshots: snapshots})
	m = updated.(Model)
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 66, Height: 20})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(Model)

	view := m.View()
	if got := lipgloss.Width(view); got != 66 {
		t.Fatalf("session picker width = %d, want 66:\n%s", got, view)
	}
	if got := lipgloss.Height(view); got != 20 {
		t.Fatalf("session picker height = %d, want 20:\n%s", got, view)
	}
	if !strings.Contains(view, "1–14 OF 30") {
		t.Fatalf("session pagination summary missing:\n%s", view)
	}
}

func TestWaitingFrameFillsPiDisplay(t *testing.T) {
	m := NewModel(fakeLoader{}, fakeMetrics{}, Config{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 66, Height: 20})
	m = updated.(Model)
	view := m.View()
	if got := lipgloss.Width(view); got != 66 {
		t.Fatalf("waiting frame width = %d, want 66:\n%s", got, view)
	}
	if got := lipgloss.Height(view); got != 20 {
		t.Fatalf("waiting frame height = %d, want 20:\n%s", got, view)
	}
	for _, want := range []string{"WAITING FOR DATA", "STATUSLINE COMMAND", "PI HEALTH", "[r] Refresh"} {
		if !strings.Contains(view, want) {
			t.Fatalf("waiting frame does not contain %q:\n%s", want, view)
		}
	}
}

func TestClearTerminalWritesEraseBeforeRenderSequence(t *testing.T) {
	var output bytes.Buffer
	if err := clearTerminal(&output); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != clearScreenSequence {
		t.Fatalf("clearTerminal() = %q, want %q", got, clearScreenSequence)
	}
}

func modelSchema() int { return statusmodel.CurrentSchemaVersion }

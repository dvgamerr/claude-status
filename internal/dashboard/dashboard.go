package dashboard

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	statusmodel "github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
)

type Config struct {
	RefreshInterval time.Duration
	StaleAfter      time.Duration
	InitialSession  string
	Inline          bool
}

type snapshotLoader interface {
	LoadAll() ([]statusmodel.Snapshot, error)
}

type metricsReader interface {
	Read() systeminfo.Stats
}

type Model struct {
	loader        snapshotLoader
	metrics       metricsReader
	config        Config
	width         int
	height        int
	snapshots     []statusmodel.Snapshot
	stats         systeminfo.Stats
	selectedID    string
	sessionCursor int
	showSessions  bool
	lastError     error
	now           func() time.Time
}

type tickMsg time.Time

type dataMsg struct {
	snapshots []statusmodel.Snapshot
	stats     systeminfo.Stats
	err       error
}

var (
	colorOrange = lipgloss.Color("#FFB86B")
	colorBlue   = lipgloss.Color("#7DD3FC")
	colorGreen  = lipgloss.Color("#6EE7A8")
	colorYellow = lipgloss.Color("#FDE68A")
	colorRed    = lipgloss.Color("#FCA5A5")
	colorMuted  = lipgloss.Color("#7E8AA6")
	colorBorder = lipgloss.Color("#34415D")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorOrange)
	labelStyle = lipgloss.NewStyle().Foreground(colorMuted)
	helpStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	errorStyle = lipgloss.NewStyle().Foreground(colorRed)
	panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(0, 1)
)

func Run(ctx context.Context, input io.Reader, output io.Writer, loader snapshotLoader, metrics metricsReader, config Config) error {
	model := NewModel(loader, metrics, config)
	options := []tea.ProgramOption{tea.WithContext(ctx), tea.WithInput(input), tea.WithOutput(output)}
	if !config.Inline {
		options = append(options, tea.WithAltScreen())
	}
	program := tea.NewProgram(model, options...)
	_, err := program.Run()
	return err
}

func NewModel(loader snapshotLoader, metrics metricsReader, config Config) Model {
	if config.RefreshInterval <= 0 {
		config.RefreshInterval = time.Second
	}
	if config.StaleAfter <= 0 {
		config.StaleAfter = 15 * time.Second
	}
	return Model{
		loader:     loader,
		metrics:    metrics,
		config:     config,
		selectedID: config.InitialSession,
		now:        time.Now,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadData(), m.nextTick())
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		return m, tea.Batch(m.loadData(), m.nextTick())
	case dataMsg:
		m.snapshots = msg.snapshots
		m.stats = msg.stats
		m.lastError = msg.err
		m.reconcileSelection()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "s":
			m.showSessions = !m.showSessions
			m.reconcileSelection()
		case "esc":
			m.showSessions = false
		case "r":
			return m, m.loadData()
		case "up", "k":
			if m.showSessions && len(m.snapshots) > 0 {
				m.sessionCursor = (m.sessionCursor - 1 + len(m.snapshots)) % len(m.snapshots)
			}
		case "down", "j":
			if m.showSessions && len(m.snapshots) > 0 {
				m.sessionCursor = (m.sessionCursor + 1) % len(m.snapshots)
			}
		case "enter":
			if m.showSessions && len(m.snapshots) > 0 {
				m.selectedID = m.snapshots[m.sessionCursor].Session.ID
				m.showSessions = false
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.showSessions {
		return m.sessionsView()
	}
	return m.dashboardView()
}

func (m Model) loadData() tea.Cmd {
	return func() tea.Msg {
		snapshots, err := m.loader.LoadAll()
		return dataMsg{snapshots: snapshots, stats: m.metrics.Read(), err: err}
	}
}

func (m Model) nextTick() tea.Cmd {
	return tea.Tick(m.config.RefreshInterval, func(at time.Time) tea.Msg { return tickMsg(at) })
}

func (m *Model) reconcileSelection() {
	if len(m.snapshots) == 0 {
		m.sessionCursor = 0
		return
	}
	for i, snapshot := range m.snapshots {
		if snapshot.Session.ID == m.selectedID {
			m.sessionCursor = i
			return
		}
	}
	m.selectedID = m.snapshots[0].Session.ID
	m.sessionCursor = 0
}

func (m Model) selectedSnapshot() (statusmodel.Snapshot, bool) {
	for _, snapshot := range m.snapshots {
		if snapshot.Session.ID == m.selectedID {
			return snapshot, true
		}
	}
	return statusmodel.Snapshot{}, false
}

func (m Model) dashboardView() string {
	width := m.contentWidth()
	now := m.now()
	header := titleStyle.Render("Claude Status") + labelStyle.Render("  "+now.Format("15:04:05 MST"))

	snapshot, found := m.selectedSnapshot()
	if !found {
		body := strings.Join([]string{
			header,
			"",
			labelStyle.Render("Waiting for Claude Code status data…"),
			"",
			"Configure statusLine to run:",
			titleStyle.Render("claude-status ingest"),
			"",
			m.systemLine(),
			"",
			helpStyle.Render("[q] Quit  [r] Refresh  [s] Sessions"),
		}, "\n")
		if m.lastError != nil {
			body += "\n" + errorStyle.Render(m.lastError.Error())
		}
		return panelStyle.Width(max(20, width-4)).Render(body) + "\n"
	}

	age := now.Sub(snapshot.CapturedAt)
	if age < 0 {
		age = 0
	}
	status := lipgloss.NewStyle().Bold(true).Foreground(colorGreen).Render("LIVE")
	if age > m.config.StaleAfter {
		status = lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render("STALE")
	}
	modelName := snapshot.Model.DisplayName
	if modelName == "" {
		modelName = snapshot.Model.ID
	}
	if modelName == "" {
		modelName = "Claude"
	}

	barWidth := min(28, max(10, width-43))
	lines := []string{
		header + "  " + status,
		"",
		field("Model", modelName),
		meterRow("5-hour", snapshot.RateLimits.FiveHour.UsedPercentage, resetText(snapshot.RateLimits.FiveHour.ResetsAt, now), barWidth),
		meterRow("Weekly", snapshot.RateLimits.SevenDay.UsedPercentage, resetText(snapshot.RateLimits.SevenDay.ResetsAt, now), barWidth),
		"",
		meterRow("Context", snapshot.Context.UsedPercentage, contextText(snapshot.Context), barWidth),
		field("Session", durationText(snapshot.Cost.TotalDurationMS)),
		field("Est. cost", costText(snapshot.Cost.TotalCostUSD)),
		field("Code", codeText(snapshot.Cost.TotalLinesAdded, snapshot.Cost.TotalLinesRemoved)),
		"",
		m.systemLine(),
		"",
		labelStyle.Render("Session") + "      " + clip(sessionLabel(snapshot), max(12, width-24)),
		labelStyle.Render("Last update") + "  " + ageText(age) + " ago",
	}
	if snapshot.Effort != "" || snapshot.ThinkingEnabled != nil {
		lines = append(lines, labelStyle.Render("Mode")+"         "+modeText(snapshot))
	}
	if m.lastError != nil {
		lines = append(lines, "", errorStyle.Render(clip(m.lastError.Error(), max(20, width-4))))
	}
	lines = append(lines, "", helpStyle.Render("[q] Quit  [r] Refresh  [s] Sessions"))

	return panelStyle.Width(max(20, width-4)).Render(strings.Join(lines, "\n")) + "\n"
}

func (m Model) sessionsView() string {
	width := m.contentWidth()
	lines := []string{titleStyle.Render("Sessions"), labelStyle.Render("Newest snapshot first"), ""}
	if len(m.snapshots) == 0 {
		lines = append(lines, "No session snapshots yet.")
	} else {
		for i, snapshot := range m.snapshots {
			cursor := "  "
			style := lipgloss.NewStyle()
			if i == m.sessionCursor {
				cursor = "› "
				style = style.Bold(true).Foreground(colorBlue)
			}
			modelName := snapshot.Model.DisplayName
			if modelName == "" {
				modelName = snapshot.Model.ID
			}
			line := fmt.Sprintf("%s%-24s %-12s %s ago", cursor, clip(sessionLabel(snapshot), 24), clip(modelName, 12), ageText(max(0, m.now().Sub(snapshot.CapturedAt))))
			lines = append(lines, style.Render(clip(line, max(20, width-4))))
		}
	}
	lines = append(lines, "", helpStyle.Render("[↑/↓] Select  [enter] Open  [s/esc] Back  [q] Quit"))
	return panelStyle.Width(max(20, width-4)).Render(strings.Join(lines, "\n")) + "\n"
}

func (m Model) systemLine() string {
	parts := []string{
		"CPU " + percentOrDash(m.stats.CPUPercent),
		"RAM " + memoryText(m.stats.MemoryUsedBytes, m.stats.MemoryTotalBytes),
		"Temp " + temperatureText(m.stats.TemperatureC),
	}
	if m.stats.Load1 != nil {
		parts = append(parts, fmt.Sprintf("Load %.2f", *m.stats.Load1))
	}
	return labelStyle.Render("Pi") + "           " + strings.Join(parts, "  ")
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 80
	}
	return min(100, max(28, m.width))
}

func meterRow(label string, percentage *float64, suffix string, width int) string {
	if suffix == "" {
		suffix = "--"
	}
	if percentage == nil {
		return fmt.Sprintf("%-12s %s  --%%  %s", label, strings.Repeat("░", width), suffix)
	}
	value := min(100, max(0, *percentage))
	filled := int(math.Round(value * float64(width) / 100))
	barStyle := lipgloss.NewStyle().Foreground(thresholdColor(value))
	bar := barStyle.Render(strings.Repeat("█", filled)) + labelStyle.Render(strings.Repeat("░", width-filled))
	return fmt.Sprintf("%-12s %s %3.0f%%  %s", label, bar, value, suffix)
}

func thresholdColor(value float64) lipgloss.Color {
	if value >= 90 {
		return colorRed
	}
	if value >= 70 {
		return colorYellow
	}
	return colorGreen
}

func field(label, value string) string {
	if value == "" {
		value = "--"
	}
	return fmt.Sprintf("%-12s %s", label, value)
}

func resetText(epoch *int64, now time.Time) string {
	if epoch == nil || *epoch <= 0 {
		return "reset --"
	}
	reset := time.Unix(*epoch, 0)
	remaining := reset.Sub(now)
	if remaining <= 0 {
		return "reset due"
	}
	clock := reset.Local().Format("15:04")
	if remaining >= 24*time.Hour {
		clock = reset.Local().Format("Mon 15:04")
	}
	return "reset " + clock + " (in " + compactDuration(remaining) + ")"
}

func contextText(context statusmodel.Context) string {
	if context.WindowSize == nil {
		return "tokens --"
	}
	var used int64
	if context.TotalInputTokens != nil {
		used += *context.TotalInputTokens
	}
	if context.TotalOutputTokens != nil {
		used += *context.TotalOutputTokens
	}
	if used == 0 && context.UsedPercentage != nil {
		used = int64(math.Round(float64(*context.WindowSize) * *context.UsedPercentage / 100))
	}
	return humanTokens(used) + " / " + humanTokens(*context.WindowSize)
}

func humanTokens(value int64) string {
	if value >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	}
	if value >= 1_000 {
		return fmt.Sprintf("%.0fk", float64(value)/1_000)
	}
	return fmt.Sprintf("%d", value)
}

func durationText(milliseconds *int64) string {
	if milliseconds == nil {
		return "--"
	}
	return formatClock(time.Duration(*milliseconds) * time.Millisecond)
}

func formatClock(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	hours := int(value / time.Hour)
	value -= time.Duration(hours) * time.Hour
	minutes := int(value / time.Minute)
	value -= time.Duration(minutes) * time.Minute
	seconds := int(value / time.Second)
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func costText(value *float64) string {
	if value == nil {
		return "--"
	}
	return fmt.Sprintf("$%.2f", *value)
}

func codeText(added, removed *int64) string {
	if added == nil && removed == nil {
		return "--"
	}
	var add, remove int64
	if added != nil {
		add = *added
	}
	if removed != nil {
		remove = *removed
	}
	return fmt.Sprintf("+%d  -%d", add, remove)
}

func percentOrDash(value *float64) string {
	if value == nil {
		return "--"
	}
	return fmt.Sprintf("%.0f%%", *value)
}

func memoryText(used, total *uint64) string {
	if used == nil || total == nil {
		return "--"
	}
	return fmt.Sprintf("%.1f/%.1f GB", float64(*used)/(1<<30), float64(*total)/(1<<30))
}

func temperatureText(value *float64) string {
	if value == nil {
		return "--"
	}
	return fmt.Sprintf("%.0f°C", *value)
}

func modeText(snapshot statusmodel.Snapshot) string {
	parts := make([]string, 0, 2)
	if snapshot.Effort != "" {
		parts = append(parts, "effort "+snapshot.Effort)
	}
	if snapshot.ThinkingEnabled != nil {
		if *snapshot.ThinkingEnabled {
			parts = append(parts, "thinking on")
		} else {
			parts = append(parts, "thinking off")
		}
	}
	return strings.Join(parts, " · ")
}

func sessionLabel(snapshot statusmodel.Snapshot) string {
	if snapshot.Session.Name != "" {
		return snapshot.Session.Name + " (" + shortID(snapshot.Session.ID) + ")"
	}
	return shortID(snapshot.Session.ID)
}

func shortID(id string) string {
	if utf8.RuneCountInString(id) <= 12 {
		return id
	}
	return clip(id, 12)
}

func ageText(age time.Duration) string {
	if age < time.Second {
		return "just now"
	}
	return compactDuration(age)
}

func compactDuration(value time.Duration) string {
	value = value.Round(time.Second)
	if value < time.Minute {
		return fmt.Sprintf("%ds", int(value/time.Second))
	}
	if value < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(value/time.Minute), int(value%time.Minute/time.Second))
	}
	if value < 24*time.Hour {
		return fmt.Sprintf("%dh%02dm", int(value/time.Hour), int(value%time.Hour/time.Minute))
	}
	return fmt.Sprintf("%dd%02dh", int(value/(24*time.Hour)), int(value%(24*time.Hour)/time.Hour))
}

func clip(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

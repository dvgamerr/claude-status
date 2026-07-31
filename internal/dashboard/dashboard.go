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
	"github.com/charmbracelet/x/ansi"
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
	bodyWidth := max(20, width-4)
	now := m.now()

	snapshot, found := m.selectedSnapshot()
	if !found {
		body := strings.Join([]string{
			betweenRendered(titleStyle.Render("✦ Claude Status"), labelStyle.Render(now.Format("15:04:05 MST")), bodyWidth),
			"",
			labelStyle.Render("Waiting for Claude Code status data…"),
			"",
			"Configure statusLine to run:",
			titleStyle.Render("claude-status ingest"),
			"",
			infoLine("PI", m.systemText(), bodyWidth),
			"",
			helpStyle.Render("[q] Quit  [r] Refresh  [s] Sessions"),
		}, "\n")
		if m.lastError != nil {
			body += "\n" + errorStyle.Render(m.lastError.Error())
		}
		return panelStyle.Width(bodyWidth + 2).Render(body)
	}

	age := now.Sub(snapshot.CapturedAt)
	if age < 0 {
		age = 0
	}
	statusText := "● LIVE"
	statusColor := colorGreen
	if age > m.config.StaleAfter {
		statusText = "● STALE"
		statusColor = colorRed
	}
	status := lipgloss.NewStyle().Bold(true).Foreground(statusColor).Render(statusText)

	var lines []string
	if bodyWidth >= 58 {
		lines = m.piDashboardLines(snapshot, now, age, status, bodyWidth)
	} else {
		lines = m.compactDashboardLines(snapshot, now, age, status, bodyWidth)
	}
	if m.lastError != nil {
		lines = append(lines, errorStyle.Render(clip(m.lastError.Error(), bodyWidth)))
	}
	lines = append(lines, "", helpStyle.Render("[q] Quit   [r] Refresh   [s] Sessions"))

	return panelStyle.Width(bodyWidth + 2).Render(strings.Join(lines, "\n"))
}

// piDashboardLines is designed around the Raspberry Pi Touch Display's
// 66-column console when TerminusBold 12x24 is used. The context card and the
// quota panel consume exactly 62 cells inside the outer panel at that size.
func (m Model) piDashboardLines(snapshot statusmodel.Snapshot, now time.Time, age time.Duration, status string, width int) []string {
	const (
		cardWidth = 20
		gapWidth  = 2
	)
	quotaWidth := width - cardWidth - gapWidth
	card := contextCard(snapshot, cardWidth)
	quota := quotaPanel(snapshot, now, status, quotaWidth)

	lines := make([]string, 0, 16)
	for i := range card {
		lines = append(lines, card[i]+strings.Repeat(" ", gapWidth)+quota[i])
	}
	lines = append(lines, "")
	lines = append(lines,
		infoLine("SESSION", sessionLabel(snapshot)+"  ·  "+durationText(snapshot.Cost.TotalDurationMS), width),
		infoLine("ACTIVITY", activityText(snapshot), width),
		infoLine("PI", m.systemText(), width),
		infoLine("UPDATED", ageText(age)+" ago  ·  "+now.Format("15:04:05 MST"), width),
	)
	return lines
}

func (m Model) compactDashboardLines(snapshot statusmodel.Snapshot, now time.Time, age time.Duration, status string, width int) []string {
	modelName := displayModelName(snapshot)
	return []string{
		betweenRendered(titleStyle.Render("✦ Claude Status"), status, width),
		field("Model", modelName),
		smallMeterRow("CTX", snapshot.Context.UsedPercentage, width),
		infoLine("TOKENS", compactTokenText(snapshot.Context), width),
		smallMeterRow("5H", snapshot.RateLimits.FiveHour.UsedPercentage, width),
		infoLine("RESET", resetCountdown(snapshot.RateLimits.FiveHour.ResetsAt, now), width),
		smallMeterRow("7D", snapshot.RateLimits.SevenDay.UsedPercentage, width),
		infoLine("RESET", resetCountdown(snapshot.RateLimits.SevenDay.ResetsAt, now), width),
		infoLine("SESSION", sessionLabel(snapshot), width),
		infoLine("PI", m.systemText(), width),
		infoLine("UPDATED", ageText(age)+" ago", width),
	}
}

func (m Model) sessionsView() string {
	width := m.contentWidth()
	bodyWidth := max(20, width-4)
	bodyHeight := 18
	if m.height > 0 {
		bodyHeight = max(6, m.height-2)
	}
	visible := max(1, bodyHeight-5)
	start, end := visibleRange(m.sessionCursor, len(m.snapshots), visible)
	summary := "Newest snapshot first"
	if len(m.snapshots) > visible {
		summary = fmt.Sprintf("Showing %d–%d of %d", start+1, end, len(m.snapshots))
	}
	lines := []string{titleStyle.Render("Sessions"), labelStyle.Render(summary), ""}
	if len(m.snapshots) == 0 {
		lines = append(lines, "No session snapshots yet.")
	} else {
		for i := start; i < end; i++ {
			snapshot := m.snapshots[i]
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
			lines = append(lines, style.Render(clip(line, bodyWidth)))
		}
	}
	lines = append(lines, "", helpStyle.Render("[↑/↓] Select  [enter] Open  [s/esc] Back  [q] Quit"))
	return panelStyle.Width(bodyWidth + 2).Render(strings.Join(lines, "\n"))
}

func (m Model) systemText() string {
	parts := []string{
		"CPU " + percentOrDash(m.stats.CPUPercent),
		"RAM " + memoryText(m.stats.MemoryUsedBytes, m.stats.MemoryTotalBytes),
		"Temp " + temperatureText(m.stats.TemperatureC),
	}
	if m.stats.Load1 != nil {
		parts = append(parts, fmt.Sprintf("Load %.2f", *m.stats.Load1))
	}
	if m.stats.Uptime != nil {
		parts = append(parts, "Up "+compactDuration(*m.stats.Uptime))
	}
	return strings.Join(parts, "  ")
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 80
	}
	return min(100, max(28, m.width))
}

func contextCard(snapshot statusmodel.Snapshot, width int) []string {
	inner := width - 2
	modelName := strings.ToUpper(displayModelName(snapshot))
	modelID := snapshot.Model.ID
	if modelID == "" || strings.EqualFold(modelID, snapshot.Model.DisplayName) {
		modelID = "CLAUDE CODE"
	}

	contextPercentage := percentageText(snapshot.Context.UsedPercentage)
	barWidth := max(4, inner-2)
	bar := renderBar(snapshot.Context.UsedPercentage, barWidth)

	return []string{
		colorBorderString("╭" + strings.Repeat("─", inner) + "╮"),
		boxLine(titleStyle.Render(" "+clip(modelName, inner-2)), inner),
		boxLine(labelStyle.Render(" "+clip(modelID, inner-2)), inner),
		boxLine(betweenRendered(labelStyle.Render(" CONTEXT"), titleStyle.Render(contextPercentage), inner), inner),
		boxLine(" "+bar+" ", inner),
		boxLine(labelStyle.Render(" "+clip(contextText(snapshot.Context), inner-2)), inner),
		colorBorderString("╰" + strings.Repeat("─", inner) + "╯"),
	}
}

func quotaPanel(snapshot statusmodel.Snapshot, now time.Time, status string, width int) []string {
	tokenLine := labelStyle.Render("TOKENS  ") +
		lipgloss.NewStyle().Bold(true).Foreground(colorBlue).Render("↑ "+tokenText(contextInputTokens(snapshot.Context))) +
		labelStyle.Render(" INPUT   ") +
		lipgloss.NewStyle().Bold(true).Foreground(colorOrange).Render("↓ "+tokenText(contextOutputTokens(snapshot.Context))) +
		labelStyle.Render(" OUTPUT")

	return []string{
		betweenRendered(titleStyle.Render("✦ Clauding…"), status, width),
		"",
		betweenRendered(labelStyle.Render("5H LIMIT"), labelStyle.Render(resetCountdown(snapshot.RateLimits.FiveHour.ResetsAt, now)), width),
		progressLine(snapshot.RateLimits.FiveHour.UsedPercentage, width),
		betweenRendered(labelStyle.Render("7 DAY LIMIT"), labelStyle.Render(resetCountdown(snapshot.RateLimits.SevenDay.ResetsAt, now)), width),
		progressLine(snapshot.RateLimits.SevenDay.UsedPercentage, width),
		clipStyled(tokenLine, width),
	}
}

func progressLine(percentage *float64, width int) string {
	percent := percentageText(percentage)
	barWidth := max(4, width-lipgloss.Width(percent)-1)
	return renderBar(percentage, barWidth) + " " + lipgloss.NewStyle().Bold(true).Foreground(thresholdColorValue(percentage)).Render(percent)
}

func smallMeterRow(label string, percentage *float64, width int) string {
	percent := percentageText(percentage)
	barWidth := max(4, width-len(label)-lipgloss.Width(percent)-2)
	return labelStyle.Render(label) + " " + renderBar(percentage, barWidth) + " " + lipgloss.NewStyle().Bold(true).Foreground(thresholdColorValue(percentage)).Render(percent)
}

func renderBar(percentage *float64, width int) string {
	if percentage == nil {
		return labelStyle.Render(strings.Repeat("░", width))
	}
	value := min(100, max(0, *percentage))
	filled := int(math.Round(value * float64(width) / 100))
	return lipgloss.NewStyle().Foreground(thresholdColor(value)).Render(strings.Repeat("█", filled)) +
		labelStyle.Render(strings.Repeat("░", width-filled))
}

func thresholdColorValue(value *float64) lipgloss.Color {
	if value == nil {
		return colorMuted
	}
	return thresholdColor(*value)
}

func percentageText(value *float64) string {
	if value == nil {
		return "--%"
	}
	return fmt.Sprintf("%.0f%%", min(100, max(0, *value)))
}

func resetCountdown(epoch *int64, now time.Time) string {
	if epoch == nil || *epoch <= 0 {
		return "reset --"
	}
	remaining := time.Unix(*epoch, 0).Sub(now)
	if remaining <= 0 {
		return "reset due"
	}
	return "resets " + compactDuration(remaining)
}

func displayModelName(snapshot statusmodel.Snapshot) string {
	if snapshot.Model.DisplayName != "" {
		return snapshot.Model.DisplayName
	}
	if snapshot.Model.ID != "" {
		return snapshot.Model.ID
	}
	return "Claude"
}

func activityText(snapshot statusmodel.Snapshot) string {
	parts := []string{
		"Cost " + costText(snapshot.Cost.TotalCostUSD),
		"Code " + codeText(snapshot.Cost.TotalLinesAdded, snapshot.Cost.TotalLinesRemoved),
	}
	if mode := modeText(snapshot); mode != "" {
		parts = append(parts, mode)
	}
	return strings.Join(parts, "  ·  ")
}

func contextInputTokens(context statusmodel.Context) *int64 {
	if context.TotalInputTokens != nil {
		return context.TotalInputTokens
	}
	usage := context.CurrentUsage
	if usage.InputTokens == nil && usage.CacheCreationInputTokens == nil && usage.CacheReadInputTokens == nil {
		return nil
	}
	value := int64(0)
	for _, part := range []*int64{usage.InputTokens, usage.CacheCreationInputTokens, usage.CacheReadInputTokens} {
		if part != nil {
			value += *part
		}
	}
	return &value
}

func contextOutputTokens(context statusmodel.Context) *int64 {
	if context.TotalOutputTokens != nil {
		return context.TotalOutputTokens
	}
	return context.CurrentUsage.OutputTokens
}

func compactTokenText(context statusmodel.Context) string {
	return "↑ " + tokenText(contextInputTokens(context)) + " input  ↓ " + tokenText(contextOutputTokens(context)) + " output"
}

func tokenText(value *int64) string {
	if value == nil {
		return "--"
	}
	return humanTokens(*value)
}

func infoLine(label, value string, width int) string {
	prefix := fmt.Sprintf("%-9s", label)
	line := labelStyle.Render(prefix) + value
	return clipStyled(line, width)
}

func boxLine(content string, inner int) string {
	return colorBorderString("│") + padStyled(content, inner) + colorBorderString("│")
}

func colorBorderString(value string) string {
	return lipgloss.NewStyle().Foreground(colorBorder).Render(value)
}

func betweenRendered(left, right string, width int) string {
	space := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", space) + right
}

func padStyled(value string, width int) string {
	return value + strings.Repeat(" ", max(0, width-lipgloss.Width(value)))
}

func clipStyled(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, "…")
}

func visibleRange(cursor, total, visible int) (int, int) {
	if total <= visible {
		return 0, total
	}
	start := cursor - visible/2
	start = min(max(0, start), total-visible)
	return start, start + visible
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
		return trimDecimal(fmt.Sprintf("%.1f", float64(value)/1_000_000)) + "M"
	}
	if value >= 1_000 {
		return trimDecimal(fmt.Sprintf("%.1f", float64(value)/1_000)) + "k"
	}
	return fmt.Sprintf("%d", value)
}

func trimDecimal(value string) string {
	return strings.TrimSuffix(value, ".0")
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

package pixelui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/dvgamerr/claude-status/internal/model"
	"github.com/dvgamerr/claude-status/internal/systeminfo"
)

const (
	Width  = 800
	Height = 480
)

var (
	backgroundTop = rgb(20, 19, 17)
	cardColor     = rgb(30, 28, 25)
	cardStrong    = rgb(39, 36, 32)
	trackColor    = rgb(60, 55, 49)
	textPrimary   = rgb(246, 242, 234)
	textSecondary = rgb(188, 178, 164)
	textFaint     = rgb(126, 117, 105)
	claudeOrange  = rgb(217, 119, 87)
	claudePeach   = rgb(235, 166, 135)
	green         = rgb(52, 211, 153)
	purple        = rgb(167, 139, 250)
	yellow        = rgb(250, 204, 21)
	red           = rgb(248, 113, 113)
)

type View struct {
	Claude       *model.Snapshot
	Codex        *model.Snapshot
	Stats        systeminfo.Stats
	Now          time.Time
	StaleAfter   time.Duration
	SessionCount int
	LoadError    error
}

type Renderer struct {
	regular12 font.Face
	regular13 font.Face
	regular14 font.Face
	regular16 font.Face
	bold13    font.Face
	bold16    font.Face
	bold18    font.Face
	bold22    font.Face
	bold30    font.Face
	bold44    font.Face
}

func NewRenderer() (*Renderer, error) {
	regular, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse regular UI font: %w", err)
	}
	bold, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse bold UI font: %w", err)
	}
	face := func(parsed *opentype.Font, size float64) (font.Face, error) {
		return opentype.NewFace(parsed, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	}
	r := &Renderer{}
	for _, target := range []struct {
		destination *font.Face
		parsed      *opentype.Font
		size        float64
	}{
		{&r.regular12, regular, 12}, {&r.regular13, regular, 13}, {&r.regular14, regular, 14},
		{&r.regular16, regular, 16}, {&r.bold13, bold, 13}, {&r.bold16, bold, 16},
		{&r.bold18, bold, 18}, {&r.bold22, bold, 22}, {&r.bold30, bold, 30}, {&r.bold44, bold, 44},
	} {
		created, err := face(target.parsed, target.size)
		if err != nil {
			return nil, fmt.Errorf("create UI font %.0f: %w", target.size, err)
		}
		*target.destination = created
	}
	return r, nil
}

func (r *Renderer) Render(view View) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, Width, Height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(backgroundTop), image.Point{}, draw.Src)
	if view.Now.IsZero() {
		view.Now = time.Now()
	}
	if view.StaleAfter <= 0 {
		view.StaleAfter = 15 * time.Second
	}
	if view.Claude == nil {
		r.renderWaiting(canvas, view)
		return canvas
	}
	r.renderDashboard(canvas, view, *view.Claude)
	return canvas
}

func (r *Renderer) renderDashboard(canvas *image.RGBA, view View, snapshot model.Snapshot) {
	accent := claudeOrange
	modelName := snapshot.Model.DisplayName
	if modelName == "" {
		modelName = snapshot.Model.ID
	}
	if modelName == "" {
		modelName = "Claude"
	}

	r.drawClaudeMark(canvas, 40, 34, 17, view.Now)
	r.text(canvas, r.bold22, textPrimary, 72, 32, "CLAUDE")
	r.text(canvas, r.regular13, textSecondary, 72, 52, "PRIMARY STATUS  •  "+strings.ToUpper(fitText(r.regular13, modelName, 245)))

	age := view.Now.Sub(snapshot.CapturedAt)
	if age < 0 {
		age = 0
	}
	liveText := "LIVE"
	liveColor := green
	if age > view.StaleAfter {
		liveText = "STALE"
		liveColor = red
	}
	fillRounded(canvas, image.Rect(642, 18, 713, 46), 14, withAlpha(liveColor, 36))
	fillCircle(canvas, 657, 32, 4, liveColor)
	r.text(canvas, r.bold13, liveColor, 668, 37, liveText)
	r.textRight(canvas, r.bold18, textPrimary, 780, 37, view.Now.Format("15:04"))

	card(canvas, image.Rect(18, 68, 782, 220), 19)
	r.text(canvas, r.bold13, textSecondary, 38, 94, "CONTEXT WINDOW")
	contextPct := snapshot.Context.UsedPercentage
	percent := percentValue(contextPct)
	r.text(canvas, r.bold44, textPrimary, 37, 148, percentLabel(contextPct))
	r.text(canvas, r.bold16, accent, 39, 172, "USED")
	r.text(canvas, r.regular14, textSecondary, 167, 128, contextFraction(snapshot.Context))
	r.text(canvas, r.regular13, textFaint, 167, 151, "active tokens / model window")

	metricChip(canvas, r, image.Rect(491, 91, 619, 153), "INPUT", tokenLabel(contextInput(snapshot.Context)), claudePeach)
	metricChip(canvas, r, image.Rect(632, 91, 761, 153), "OUTPUT", tokenLabel(contextOutput(snapshot.Context)), purple)
	progress(canvas, image.Rect(38, 190, 762, 205), percent, thresholdColor(percent, accent))

	r.quotaCard(canvas, image.Rect(18, 234, 393, 342), "5 HOUR", snapshot.RateLimits.FiveHour, snapshot.RateLimits, claudeOrange, view.Now)
	r.quotaCard(canvas, image.Rect(407, 234, 782, 342), "7 DAY", snapshot.RateLimits.SevenDay, snapshot.RateLimits, claudePeach, view.Now)

	r.claudeSessionCard(canvas, image.Rect(18, 356, 526, 446), snapshot, view.Stats, age)
	r.codexCard(canvas, image.Rect(540, 356, 782, 446), view.Codex)

	footer := fmt.Sprintf("AUTO  •  %d SESSION", max(1, view.SessionCount))
	if view.SessionCount != 1 {
		footer += "S"
	}
	if view.LoadError != nil {
		footer = "STATE WARNING  •  " + fitText(r.regular12, view.LoadError.Error(), 500)
	}
	r.text(canvas, r.bold13, accent, 21, 469, footer)
	r.textRight(canvas, r.regular12, textFaint, 779, 469, "CLAUDE PRIMARY  •  800×480")
}

func (r *Renderer) renderWaiting(canvas *image.RGBA, view View) {
	accent := claudeOrange
	r.drawClaudeMark(canvas, 40, 34, 17, view.Now)
	r.text(canvas, r.bold22, textPrimary, 72, 32, "CLAUDE")
	r.text(canvas, r.regular13, textSecondary, 72, 52, "PRIMARY STATUS")
	r.textRight(canvas, r.bold18, textPrimary, 780, 37, view.Now.Format("15:04"))

	card(canvas, image.Rect(18, 68, 782, 342), 22)
	r.drawClaudeMark(canvas, 400, 145, 31, view.Now)
	r.textCentered(canvas, r.bold30, textPrimary, 400, 220, "WAITING FOR CLAUDE")
	r.textCentered(canvas, r.regular16, textSecondary, 400, 253, "Start or continue a Claude Code session on the source machine")
	progress(canvas, image.Rect(244, 289, 556, 299), animationPercent(view.Now), accent)
	r.textCentered(canvas, r.regular13, textFaint, 400, 325, "The next statusLine event will update this display")
	r.codexCard(canvas, image.Rect(540, 356, 782, 446), view.Codex)
	miniCard(canvas, image.Rect(18, 356, 526, 446))
	r.text(canvas, r.bold13, textSecondary, 36, 381, "PI HEALTH")
	r.text(canvas, r.bold18, textPrimary, 36, 411, healthPrimary(view.Stats))
	r.text(canvas, r.regular13, textFaint, 36, 434, healthSecondary(view.Stats))
	r.text(canvas, r.bold13, accent, 21, 469, "CLAUDE PRIMARY  •  WAITING")
	r.textRight(canvas, r.regular12, textFaint, 779, 469, "FRAMEBUFFER 800×480")
}

func (r *Renderer) claudeSessionCard(canvas *image.RGBA, bounds image.Rectangle, snapshot model.Snapshot, stats systeminfo.Stats, age time.Duration) {
	miniCard(canvas, bounds)
	r.text(canvas, r.bold13, claudePeach, bounds.Min.X+18, bounds.Min.Y+25, "CURRENT CLAUDE SESSION")
	r.text(canvas, r.bold18, textPrimary, bounds.Min.X+18, bounds.Min.Y+54, fitText(r.bold18, sessionName(snapshot), bounds.Dx()-36))
	details := strings.ToUpper(modelLabel(snapshot)) + "  •  " + modePrimary(snapshot) + "  •  " + ageText(age)
	r.text(canvas, r.regular12, textSecondary, bounds.Min.X+18, bounds.Min.Y+75, fitText(r.regular12, details, bounds.Dx()-36))
	r.textRight(canvas, r.regular12, textFaint, bounds.Max.X-18, bounds.Min.Y+25, healthCompact(stats))
}

func (r *Renderer) codexCard(canvas *image.RGBA, bounds image.Rectangle, snapshot *model.Snapshot) {
	miniCard(canvas, bounds)
	r.text(canvas, r.bold13, green, bounds.Min.X+18, bounds.Min.Y+24, "CODEX")
	r.textRight(canvas, r.regular12, textFaint, bounds.Max.X-17, bounds.Min.Y+24, "CURRENT SESSION")
	if snapshot == nil {
		r.text(canvas, r.bold16, textPrimary, bounds.Min.X+18, bounds.Min.Y+53, "NO SESSION")
		r.text(canvas, r.regular12, textFaint, bounds.Min.X+18, bounds.Min.Y+75, "context unavailable")
		return
	}
	r.text(canvas, r.bold16, textPrimary, bounds.Min.X+18, bounds.Min.Y+51, fitText(r.bold16, sessionName(*snapshot), bounds.Dx()-36))
	label := "CONTEXT  " + percentLabel(snapshot.Context.UsedPercentage)
	r.text(canvas, r.bold13, textSecondary, bounds.Min.X+18, bounds.Min.Y+75, label)
	progress(canvas, image.Rect(bounds.Min.X+116, bounds.Min.Y+66, bounds.Max.X-18, bounds.Min.Y+77), percentValue(snapshot.Context.UsedPercentage), green)
}

func (r *Renderer) quotaCard(canvas *image.RGBA, bounds image.Rectangle, label string, window model.RateWindow, limits model.RateLimits, accent color.RGBA, now time.Time) {
	card(canvas, bounds, 17)
	r.text(canvas, r.bold13, textSecondary, bounds.Min.X+18, bounds.Min.Y+25, label+" LIMIT")
	if window.UsedPercentage == nil && limits.Unlimited != nil && *limits.Unlimited {
		r.text(canvas, r.bold22, green, bounds.Min.X+18, bounds.Min.Y+61, "UNMETERED")
		plan := strings.ToUpper(strings.ReplaceAll(limits.Plan, "_", " "))
		if plan == "" {
			plan = "ACCOUNT HAS NO METERED LIMIT"
		}
		r.text(canvas, r.regular12, textFaint, bounds.Min.X+18, bounds.Min.Y+83, fitText(r.regular12, plan, bounds.Dx()-36))
		progress(canvas, image.Rect(bounds.Min.X+18, bounds.Max.Y-20, bounds.Max.X-18, bounds.Max.Y-10), 100, green)
		return
	}
	pct := percentValue(window.UsedPercentage)
	r.text(canvas, r.bold30, textPrimary, bounds.Min.X+18, bounds.Min.Y+64, percentLabel(window.UsedPercentage))
	r.text(canvas, r.regular13, textSecondary, bounds.Min.X+112, bounds.Min.Y+55, resetLabel(window.ResetsAt, now))
	progress(canvas, image.Rect(bounds.Min.X+18, bounds.Max.Y-20, bounds.Max.X-18, bounds.Max.Y-10), pct, thresholdColor(pct, accent))
}

func metricChip(canvas *image.RGBA, r *Renderer, bounds image.Rectangle, label, value string, accent color.RGBA) {
	fillRounded(canvas, bounds, 13, cardStrong)
	r.text(canvas, r.bold13, accent, bounds.Min.X+14, bounds.Min.Y+22, label)
	r.text(canvas, r.bold18, textPrimary, bounds.Min.X+14, bounds.Min.Y+48, value)
}

func card(canvas *image.RGBA, bounds image.Rectangle, radius int) {
	fillRounded(canvas, bounds.Add(image.Pt(0, 3)), radius, rgb(4, 7, 14))
	fillRounded(canvas, bounds, radius, cardColor)
}

func miniCard(canvas *image.RGBA, bounds image.Rectangle) {
	fillRounded(canvas, bounds, 14, cardColor)
}

func progress(canvas *image.RGBA, bounds image.Rectangle, percentage float64, accent color.RGBA) {
	fillRounded(canvas, bounds, bounds.Dy()/2, trackColor)
	percentage = min(100, max(0, percentage))
	fillWidth := int(math.Round(float64(bounds.Dx()) * percentage / 100))
	if fillWidth > 0 {
		fillRounded(canvas, image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+min(bounds.Dx(), max(bounds.Dy(), fillWidth)), bounds.Max.Y), bounds.Dy()/2, accent)
	}
}

func fillRounded(canvas *image.RGBA, bounds image.Rectangle, radius int, fill color.RGBA) {
	if bounds.Empty() {
		return
	}
	radius = min(radius, min(bounds.Dx()/2, bounds.Dy()/2))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		inset := 0
		if radius > 0 {
			var dy float64
			switch {
			case y < bounds.Min.Y+radius:
				dy = float64(bounds.Min.Y+radius-y) - 0.5
			case y >= bounds.Max.Y-radius:
				dy = float64(y-(bounds.Max.Y-radius-1)) - 0.5
			}
			if dy > 0 {
				inset = radius - int(math.Sqrt(max(0, float64(radius*radius)-dy*dy)))
			}
		}
		draw.Draw(canvas, image.Rect(bounds.Min.X+inset, y, bounds.Max.X-inset, y+1), image.NewUniform(fill), image.Point{}, draw.Src)
	}
}

func fillCircle(canvas *image.RGBA, centerX, centerY, radius int, fill color.RGBA) {
	fillRounded(canvas, image.Rect(centerX-radius, centerY-radius, centerX+radius, centerY+radius), radius, fill)
}

func (r *Renderer) drawClaudeMark(canvas *image.RGBA, centerX, centerY, radius int, now time.Time) {
	fillCircle(canvas, centerX, centerY, radius+8, rgb(48, 34, 28))
	phase := float64(now.UnixMilli()%2000) / 2000 * 2 * math.Pi
	for ray := 0; ray < 8; ray++ {
		angle := float64(ray)*math.Pi/4 + phase/8
		pulse := (math.Sin(phase+float64(ray)*math.Pi/4) + 1) / 2
		inner := float64(radius) * 0.28
		outer := float64(radius) * (0.73 + pulse*0.18)
		fill := claudeOrange
		if pulse > 0.72 {
			fill = claudePeach
		}
		drawThickLine(
			canvas,
			centerX+int(math.Cos(angle)*inner),
			centerY+int(math.Sin(angle)*inner),
			centerX+int(math.Cos(angle)*outer),
			centerY+int(math.Sin(angle)*outer),
			max(2, radius/6),
			fill,
		)
	}
	fillCircle(canvas, centerX, centerY, max(3, radius/5), claudeOrange)
	orbit := phase * 1.5
	fillCircle(
		canvas,
		centerX+int(math.Cos(orbit)*float64(radius+4)),
		centerY+int(math.Sin(orbit)*float64(radius+4)),
		max(2, radius/9),
		claudePeach,
	)
}

func drawThickLine(canvas *image.RGBA, x0, y0, x1, y1, thickness int, fill color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	steps := max(max(dx, -dx), max(dy, -dy))
	if steps == 0 {
		fillCircle(canvas, x0, y0, thickness, fill)
		return
	}
	for step := 0; step <= steps; step++ {
		x := x0 + dx*step/steps
		y := y0 + dy*step/steps
		fillCircle(canvas, x, y, thickness, fill)
	}
}

func animationPercent(now time.Time) float64 {
	phase := float64(now.UnixMilli()%2000) / 2000 * 2 * math.Pi
	return 35 + (math.Sin(phase)+1)*15
}

func (r *Renderer) text(canvas *image.RGBA, face font.Face, fill color.RGBA, x, baseline int, value string) {
	drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(fill), Face: face, Dot: fixed.P(x, baseline)}
	drawer.DrawString(value)
}

func (r *Renderer) textRight(canvas *image.RGBA, face font.Face, fill color.RGBA, right, baseline int, value string) {
	r.text(canvas, face, fill, right-font.MeasureString(face, value).Ceil(), baseline, value)
}

func (r *Renderer) textCentered(canvas *image.RGBA, face font.Face, fill color.RGBA, center, baseline int, value string) {
	r.text(canvas, face, fill, center-font.MeasureString(face, value).Ceil()/2, baseline, value)
}

func fitText(face font.Face, value string, maxWidth int) string {
	value = strings.TrimSpace(value)
	if font.MeasureString(face, value).Ceil() <= maxWidth {
		return value
	}
	runes := []rune(value)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + "…"
		if font.MeasureString(face, candidate).Ceil() <= maxWidth {
			return candidate
		}
	}
	return "…"
}

func providerName(provider string) string {
	if strings.EqualFold(strings.TrimSpace(provider), "codex") {
		return "CODEX"
	}
	return "CLAUDE"
}

func percentValue(value *float64) float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return 0
	}
	return min(100, max(0, *value))
}

func percentLabel(value *float64) string {
	if value == nil {
		return "--"
	}
	return fmt.Sprintf("%.0f%%", percentValue(value))
}

func thresholdColor(value float64, fallback color.RGBA) color.RGBA {
	switch {
	case value >= 90:
		return red
	case value >= 70:
		return yellow
	default:
		return fallback
	}
}

func contextInput(context model.Context) *int64 {
	if context.TotalInputTokens != nil {
		return context.TotalInputTokens
	}
	parts := []*int64{context.CurrentUsage.InputTokens, context.CurrentUsage.CacheCreationInputTokens, context.CurrentUsage.CacheReadInputTokens}
	var total int64
	found := false
	for _, part := range parts {
		if part != nil {
			total += *part
			found = true
		}
	}
	if !found {
		return nil
	}
	return &total
}

func contextOutput(context model.Context) *int64 {
	if context.TotalOutputTokens != nil {
		return context.TotalOutputTokens
	}
	return context.CurrentUsage.OutputTokens
}

func contextFraction(context model.Context) string {
	if context.WindowSize == nil {
		return "token window unavailable"
	}
	used := int64(0)
	if input := contextInput(context); input != nil {
		used += *input
	}
	if output := contextOutput(context); output != nil {
		used += *output
	}
	if used == 0 && context.UsedPercentage != nil {
		used = int64(math.Round(float64(*context.WindowSize) * percentValue(context.UsedPercentage) / 100))
	}
	return tokenLabel(&used) + " / " + tokenLabel(context.WindowSize)
}

func tokenLabel(value *int64) string {
	if value == nil {
		return "--"
	}
	switch {
	case *value >= 1_000_000:
		return trimZero(fmt.Sprintf("%.1fM", float64(*value)/1_000_000))
	case *value >= 1_000:
		return trimZero(fmt.Sprintf("%.1fK", float64(*value)/1_000))
	default:
		return fmt.Sprintf("%d", *value)
	}
}

func trimZero(value string) string {
	value = strings.Replace(value, ".0M", "M", 1)
	return strings.Replace(value, ".0K", "K", 1)
}

func resetLabel(epoch *int64, now time.Time) string {
	if epoch == nil || *epoch <= 0 {
		return "reset unavailable"
	}
	remaining := time.Unix(*epoch, 0).Sub(now)
	if remaining <= 0 {
		return "reset due"
	}
	return "resets in " + durationLabel(remaining)
}

func durationLabel(value time.Duration) string {
	if value < time.Minute {
		return fmt.Sprintf("%ds", int(value/time.Second))
	}
	if value < time.Hour {
		return fmt.Sprintf("%dm", int(value/time.Minute))
	}
	if value < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(value/time.Hour), int(value%time.Hour/time.Minute))
	}
	return fmt.Sprintf("%dd %dh", int(value/(24*time.Hour)), int(value%(24*time.Hour)/time.Hour))
}

func ageText(value time.Duration) string {
	if value < time.Second {
		return "just now"
	}
	return durationLabel(value) + " ago"
}

func sessionName(snapshot model.Snapshot) string {
	if snapshot.Session.Name != "" {
		return snapshot.Session.Name
	}
	id := snapshot.Session.ID
	if len(id) > 18 {
		id = id[:18] + "…"
	}
	return id
}

func modelLabel(snapshot model.Snapshot) string {
	if snapshot.Model.DisplayName != "" {
		return snapshot.Model.DisplayName
	}
	if snapshot.Model.ID != "" {
		return snapshot.Model.ID
	}
	return providerName(snapshot.Provider)
}

func healthPrimary(stats systeminfo.Stats) string {
	return "CPU " + floatLabel(stats.CPUPercent, "%.0f%%") + "   RAM " + memoryLabel(stats.MemoryUsedBytes, stats.MemoryTotalBytes)
}

func healthSecondary(stats systeminfo.Stats) string {
	parts := []string{"TEMP " + floatLabel(stats.TemperatureC, "%.0f°C")}
	if stats.Load1 != nil {
		parts = append(parts, fmt.Sprintf("LOAD %.2f", *stats.Load1))
	}
	return strings.Join(parts, "   ")
}

func healthCompact(stats systeminfo.Stats) string {
	return "PI  " + floatLabel(stats.CPUPercent, "%.0f%%") + "  •  " + floatLabel(stats.TemperatureC, "%.0f°C")
}

func floatLabel(value *float64, format string) string {
	if value == nil {
		return "--"
	}
	return fmt.Sprintf(format, *value)
}

func memoryLabel(used, total *uint64) string {
	if used == nil || total == nil {
		return "--"
	}
	return fmt.Sprintf("%.1f/%.1fG", float64(*used)/(1<<30), float64(*total)/(1<<30))
}

func modePrimary(snapshot model.Snapshot) string {
	if snapshot.Effort != "" {
		return strings.ToUpper(snapshot.Effort) + " EFFORT"
	}
	if snapshot.ThinkingEnabled != nil && *snapshot.ThinkingEnabled {
		return "THINKING ON"
	}
	return "STANDARD"
}

func rgb(red, green, blue uint8) color.RGBA {
	return color.RGBA{R: red, G: green, B: blue, A: 255}
}

func withAlpha(value color.RGBA, alpha uint8) color.RGBA {
	base := cardColor
	factor := float64(alpha) / 255
	return rgb(
		uint8(float64(value.R)*factor+float64(base.R)*(1-factor)),
		uint8(float64(value.G)*factor+float64(base.G)*(1-factor)),
		uint8(float64(value.B)*factor+float64(base.B)*(1-factor)),
	)
}

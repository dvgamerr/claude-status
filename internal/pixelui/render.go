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

// activityStaleAfter bounds how long a hook-reported working/waiting state is
// trusted. If the source machine crashes or a Stop hook is missed, the
// mascot falls back to idle instead of animating "working" forever.
const activityStaleAfter = 10 * time.Minute

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

// Layout: an 18px page margin, a fixed-width status rail (mascot + activity
// state + Pi health) on the left, and a metrics column on the right. The
// rail is the visual anchor on every screen — Nielsen's "visibility of
// system status" heuristic calls for the mascot's animation and any pending
// approval to be the first thing noticed, not buried in a corner icon.
const (
	pageMargin     = 18
	railLeft       = pageMargin
	railRight      = 232
	contentLeft    = 246
	contentRight   = Width - pageMargin
	sectionsTop    = 68
	sectionsBottom = 446
	footerBaseline = 469
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
	activity := resolveActivity(view.Claude, view.Now)
	if view.Claude == nil {
		r.renderWaiting(canvas, view, activity)
		return canvas
	}
	r.renderDashboard(canvas, view, *view.Claude, activity)
	return canvas
}

func (r *Renderer) renderDashboard(canvas *image.RGBA, view View, snapshot model.Snapshot, activity string) {
	accent := claudeOrange
	modelName := snapshot.Model.DisplayName
	if modelName == "" {
		modelName = snapshot.Model.ID
	}

	r.renderHeader(canvas, view, modelName)
	r.renderRail(canvas, image.Rect(railLeft, sectionsTop, railRight, sectionsBottom), activity, &snapshot, view.Stats, view.Now)

	card(canvas, image.Rect(contentLeft, 68, contentRight, 220), 19)
	r.text(canvas, r.bold13, textSecondary, contentLeft+20, 92, "CONTEXT WINDOW")
	contextPct := snapshot.Context.UsedPercentage
	percent := percentValue(contextPct)
	r.text(canvas, r.bold44, textPrimary, contentLeft+19, 146, percentLabel(contextPct))
	r.text(canvas, r.bold16, accent, contentLeft+21, 170, "USED")
	r.text(canvas, r.regular13, textSecondary, contentLeft+150, 146, fitText(r.regular13, contextFraction(snapshot.Context), 210))

	metricChip(canvas, r, image.Rect(contentLeft+228, 88, contentRight-152, 150), "INPUT", tokenLabel(contextInput(snapshot.Context)), claudePeach)
	metricChip(canvas, r, image.Rect(contentRight-137, 88, contentRight-19, 150), "OUTPUT", tokenLabel(contextOutput(snapshot.Context)), purple)
	progress(canvas, image.Rect(contentLeft+20, 190, contentRight-19, 203), percent, thresholdColor(percent, accent))

	quotaWidth := (contentRight - contentLeft - 14) / 2
	r.quotaCard(canvas, image.Rect(contentLeft, 234, contentLeft+quotaWidth, 342), "5 HOUR", snapshot.RateLimits.FiveHour, snapshot.RateLimits, claudeOrange, view.Now)
	r.quotaCard(canvas, image.Rect(contentLeft+quotaWidth+14, 234, contentRight, 342), "7 DAY", snapshot.RateLimits.SevenDay, snapshot.RateLimits, claudePeach, view.Now)

	bottomWidth := (contentRight - contentLeft - 14) / 2
	r.claudeSessionCard(canvas, image.Rect(contentLeft, 356, contentLeft+bottomWidth, 446), snapshot)
	r.codexCard(canvas, image.Rect(contentLeft+bottomWidth+14, 356, contentRight, 446), view.Codex)

	footer := fmt.Sprintf("AUTO  •  %d SESSION", max(1, view.SessionCount))
	if view.SessionCount != 1 {
		footer += "S"
	}
	if view.LoadError != nil {
		footer = "STATE WARNING  •  " + fitText(r.regular12, view.LoadError.Error(), 500)
	}
	r.text(canvas, r.bold13, accent, 21, footerBaseline, footer)
	r.textRight(canvas, r.regular12, textFaint, contentRight, footerBaseline, "CLAUDE PRIMARY  •  800×480")
}

func (r *Renderer) renderWaiting(canvas *image.RGBA, view View, activity string) {
	r.renderHeader(canvas, view, "")
	r.renderRail(canvas, image.Rect(railLeft, sectionsTop, railRight, sectionsBottom), activity, nil, view.Stats, view.Now)

	card(canvas, image.Rect(contentLeft, 68, contentRight, 342), 19)
	center := contentLeft + (contentRight-contentLeft)/2
	r.textCentered(canvas, r.bold30, textPrimary, center, 190, "WAITING FOR CLAUDE")
	r.textCentered(canvas, r.regular16, textSecondary, center, 223, "Start or continue a Claude Code session on the source machine")
	progress(canvas, image.Rect(center-158, 259, center+158, 269), animationPercent(view.Now), claudeOrange)
	r.textCentered(canvas, r.regular13, textFaint, center, 295, "The next statusLine event will update this display")

	bottomWidth := (contentRight - contentLeft - 14) / 2
	miniCard(canvas, image.Rect(contentLeft, 356, contentLeft+bottomWidth, 446))
	r.text(canvas, r.bold13, textSecondary, contentLeft+18, 381, "CLAUDE SESSION")
	r.text(canvas, r.bold16, textPrimary, contentLeft+18, 411, "NO SESSION YET")
	r.text(canvas, r.regular12, textFaint, contentLeft+18, 434, "status will appear once Claude Code reports in")
	r.codexCard(canvas, image.Rect(contentLeft+bottomWidth+14, 356, contentRight, 446), view.Codex)

	r.text(canvas, r.bold13, claudeOrange, 21, footerBaseline, "CLAUDE PRIMARY  •  WAITING")
	r.textRight(canvas, r.regular12, textFaint, contentRight, footerBaseline, "FRAMEBUFFER 800×480")
}

func (r *Renderer) renderHeader(canvas *image.RGBA, view View, modelName string) {
	fillCircle(canvas, 33, 36, 7, claudeOrange)
	r.text(canvas, r.bold22, textPrimary, 54, 40, "CLAUDE")
	subtitle := "PRIMARY STATUS"
	if modelName != "" {
		subtitle += "  •  " + strings.ToUpper(fitText(r.regular13, modelName, 200))
	}
	r.text(canvas, r.regular13, textSecondary, 54, 57, subtitle)

	if view.Claude != nil {
		age := view.Now.Sub(view.Claude.CapturedAt)
		if age < 0 {
			age = 0
		}
		liveText, liveColor := "LIVE", green
		if age > view.StaleAfter {
			liveText, liveColor = "STALE", red
		}
		fillRounded(canvas, image.Rect(642, 18, 713, 46), 14, withAlpha(liveColor, 36))
		fillCircle(canvas, 657, 32, 4, liveColor)
		r.text(canvas, r.bold13, liveColor, 668, 37, liveText)
	}
	r.textRight(canvas, r.bold18, textPrimary, contentRight, 37, view.Now.Format("15:04"))
}

// renderRail is the dashboard's focal point: a large animated mascot whose
// motion communicates whether the source session is working, idle, or
// blocked on a permission prompt, so that state is legible from across a
// room without reading any text.
func (r *Renderer) renderRail(canvas *image.RGBA, bounds image.Rectangle, activityState string, snapshot *model.Snapshot, stats systeminfo.Stats, now time.Time) {
	if activityState == model.ActivityWaitingApproval {
		fillRounded(canvas, bounds.Inset(-3), 23, withAlpha(yellow, 70))
	}
	card(canvas, bounds, 20)

	centerX := bounds.Min.X + bounds.Dx()/2
	mascotY := bounds.Min.Y + 100
	radius := 54
	r.drawMascot(canvas, centerX, mascotY, radius, now, activityState)
	if activityState == model.ActivityWaitingApproval {
		r.drawApprovalBadge(canvas, centerX+int(float64(radius)*0.7), mascotY-int(float64(radius)*0.7), now)
	}

	label, accent := activityLabel(activityState)
	pillWidth := bounds.Dx() - 40
	pillTop := mascotY + radius + 22
	pillBounds := image.Rect(centerX-pillWidth/2, pillTop, centerX+pillWidth/2, pillTop+30)
	fillRounded(canvas, pillBounds, 15, withAlpha(accent, 40))
	fillCircle(canvas, centerX-pillWidth/2+16, pillTop+15, 5, accent)
	r.textCentered(canvas, r.bold13, accent, centerX+8, pillTop+20, label)

	captionTop := pillBounds.Max.Y + 20
	if snapshot != nil {
		r.textCentered(canvas, r.regular13, textSecondary, centerX, captionTop, fitText(r.regular13, sessionName(*snapshot), bounds.Dx()-24))
		r.textCentered(canvas, r.regular12, textFaint, centerX, captionTop+19, activityCaption(*snapshot, activityState, now))
	} else {
		r.textCentered(canvas, r.regular13, textFaint, centerX, captionTop, "no active session")
	}

	dividerY := captionTop + 38
	fillRounded(canvas, image.Rect(bounds.Min.X+24, dividerY, bounds.Max.X-24, dividerY+2), 1, trackColor)

	healthTop := dividerY + 26
	r.text(canvas, r.bold13, textSecondary, bounds.Min.X+22, healthTop, "PI HEALTH")
	r.text(canvas, r.bold16, textPrimary, bounds.Min.X+22, healthTop+26, healthPrimary(stats))
	r.text(canvas, r.regular12, textFaint, bounds.Min.X+22, healthTop+48, healthSecondary(stats))
}

func (r *Renderer) claudeSessionCard(canvas *image.RGBA, bounds image.Rectangle, snapshot model.Snapshot) {
	miniCard(canvas, bounds)
	r.text(canvas, r.bold13, claudePeach, bounds.Min.X+18, bounds.Min.Y+25, "CLAUDE SESSION")
	r.text(canvas, r.bold18, textPrimary, bounds.Min.X+18, bounds.Min.Y+54, fitText(r.bold18, sessionName(snapshot), bounds.Dx()-36))
	details := strings.ToUpper(modelLabel(snapshot)) + "  •  " + modePrimary(snapshot)
	r.text(canvas, r.regular12, textSecondary, bounds.Min.X+18, bounds.Min.Y+75, fitText(r.regular12, details, bounds.Dx()-36))
}

func (r *Renderer) codexCard(canvas *image.RGBA, bounds image.Rectangle, snapshot *model.Snapshot) {
	miniCard(canvas, bounds)
	r.text(canvas, r.bold13, green, bounds.Min.X+18, bounds.Min.Y+24, "CODEX")
	r.textRight(canvas, r.regular12, textFaint, bounds.Max.X-17, bounds.Min.Y+24, "SESSION")
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
	r.text(canvas, r.regular13, textSecondary, bounds.Min.X+112, bounds.Min.Y+55, fitText(r.regular13, resetLabel(window.ResetsAt, now), bounds.Dx()-130))
	progress(canvas, image.Rect(bounds.Min.X+18, bounds.Max.Y-20, bounds.Max.X-18, bounds.Max.Y-10), pct, thresholdColor(pct, accent))
}

func metricChip(canvas *image.RGBA, r *Renderer, bounds image.Rectangle, label, value string, accent color.RGBA) {
	fillRounded(canvas, bounds, 13, cardStrong)
	r.text(canvas, r.bold13, accent, bounds.Min.X+14, bounds.Min.Y+22, label)
	r.text(canvas, r.bold18, textPrimary, bounds.Min.X+14, bounds.Min.Y+48, fitText(r.bold18, value, bounds.Dx()-24))
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

// activityVisual is the animation "voice" for one activity state: how fast
// the mascot pulses, how far its rays swing, and its color mood.
type activityVisual struct {
	period    time.Duration
	rayColor  color.RGBA
	rayColor2 color.RGBA
	halo      color.RGBA
	pulseAmp  float64
	orbit     bool
}

func visualForActivity(state string) activityVisual {
	switch state {
	case model.ActivityWorking:
		return activityVisual{period: 900 * time.Millisecond, rayColor: claudeOrange, rayColor2: claudePeach, halo: rgb(58, 40, 32), pulseAmp: 0.30, orbit: true}
	case model.ActivityWaitingApproval:
		return activityVisual{period: 1500 * time.Millisecond, rayColor: yellow, rayColor2: rgb(255, 226, 143), halo: rgb(58, 49, 24), pulseAmp: 0.16, orbit: false}
	default:
		return activityVisual{period: 4200 * time.Millisecond, rayColor: withAlpha(claudeOrange, 150), rayColor2: withAlpha(claudePeach, 120), halo: rgb(38, 34, 30), pulseAmp: 0.08, orbit: false}
	}
}

func activityLabel(state string) (string, color.RGBA) {
	switch state {
	case model.ActivityWorking:
		return "WORKING", claudeOrange
	case model.ActivityWaitingApproval:
		return "NEEDS APPROVAL", yellow
	default:
		return "IDLE", textSecondary
	}
}

// resolveActivity turns a possibly-stale, possibly-absent Activity field
// into the state the mascot should animate right now. When no snapshot
// exists, or when hooks haven't been installed yet, it degrades gracefully
// instead of leaving the mascot in a stuck or undefined state.
func resolveActivity(snapshot *model.Snapshot, now time.Time) string {
	if snapshot == nil {
		return model.ActivityIdle
	}
	if snapshot.Activity.State != "" {
		age := now.Sub(snapshot.Activity.UpdatedAt)
		if age >= 0 && age <= activityStaleAfter {
			return snapshot.Activity.State
		}
		return model.ActivityIdle
	}
	age := now.Sub(snapshot.CapturedAt)
	if age >= 0 && age < 3*time.Second {
		return model.ActivityWorking
	}
	return model.ActivityIdle
}

func activityCaption(snapshot model.Snapshot, state string, now time.Time) string {
	if snapshot.Activity.State == "" || snapshot.Activity.UpdatedAt.IsZero() {
		return "waiting for the next hook event"
	}
	elapsed := now.Sub(snapshot.Activity.UpdatedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	switch state {
	case model.ActivityWorking:
		return "working for " + durationLabel(elapsed)
	case model.ActivityWaitingApproval:
		return "waiting " + durationLabel(elapsed)
	default:
		return "idle for " + durationLabel(elapsed)
	}
}

func (r *Renderer) drawMascot(canvas *image.RGBA, centerX, centerY, radius int, now time.Time, state string) {
	visual := visualForActivity(state)
	fillCircle(canvas, centerX, centerY, radius+10, visual.halo)
	periodMS := visual.period.Milliseconds()
	if periodMS <= 0 {
		periodMS = 2000
	}
	phase := float64(now.UnixMilli()%periodMS) / float64(periodMS) * 2 * math.Pi
	for ray := 0; ray < 8; ray++ {
		angle := float64(ray)*math.Pi/4 + phase/8
		pulse := (math.Sin(phase+float64(ray)*math.Pi/4) + 1) / 2
		inner := float64(radius) * 0.28
		outer := float64(radius) * (0.68 + pulse*visual.pulseAmp*2)
		fill := visual.rayColor
		if pulse > 0.72 {
			fill = visual.rayColor2
		}
		drawThickLine(
			canvas,
			centerX+int(math.Cos(angle)*inner),
			centerY+int(math.Sin(angle)*inner),
			centerX+int(math.Cos(angle)*outer),
			centerY+int(math.Sin(angle)*outer),
			max(2, radius/7),
			fill,
		)
	}
	fillCircle(canvas, centerX, centerY, max(3, radius/5), visual.rayColor)
	if visual.orbit {
		orbit := phase * 1.5
		fillCircle(
			canvas,
			centerX+int(math.Cos(orbit)*float64(radius+6)),
			centerY+int(math.Sin(orbit)*float64(radius+6)),
			max(2, radius/9),
			visual.rayColor2,
		)
	}
}

// drawApprovalBadge overlays a pulsing "?" on the mascot so a pending
// permission prompt is visible at a glance, matching Nielsen's visibility
// of system status heuristic.
func (r *Renderer) drawApprovalBadge(canvas *image.RGBA, x, y int, now time.Time) {
	phase := float64(now.UnixMilli()%1100) / 1100 * 2 * math.Pi
	radius := int(15 * (1 + 0.12*math.Sin(phase)))
	fillCircle(canvas, x, y, radius+4, backgroundTop)
	fillCircle(canvas, x, y, radius, yellow)
	label := "?"
	width := font.MeasureString(r.bold16, label).Ceil()
	r.text(canvas, r.bold16, rgb(46, 36, 8), x-width/2, y+6, label)
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
		return "window unavailable"
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
	return "CPU " + floatLabel(stats.CPUPercent, "%.0f%%") + "  RAM " + memoryLabel(stats.MemoryUsedBytes, stats.MemoryTotalBytes)
}

func healthSecondary(stats systeminfo.Stats) string {
	parts := []string{"TEMP " + floatLabel(stats.TemperatureC, "%.0f°C")}
	if stats.Load1 != nil {
		parts = append(parts, fmt.Sprintf("LOAD %.2f", *stats.Load1))
	}
	return strings.Join(parts, "   ")
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

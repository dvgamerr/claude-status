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
	"github.com/dvgamerr/claude-status/internal/touch"
)

const (
	Width  = 800
	Height = 480
)

// activityStaleAfter bounds how long a hook-reported working/waiting state is
// trusted. If the source machine crashes or a Stop hook is missed, the
// mascot falls back to idle instead of animating "working" forever.
const activityStaleAfter = 10 * time.Minute

// touchRippleLifetime is how long a tap's ripple stays visible — this is
// the official Raspberry Pi touch display, so a tap should visibly answer
// back even though the dashboard has no other interactive controls.
const touchRippleLifetime = 500 * time.Millisecond

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
	sectionsTop    = 78
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
	Touches      []touch.Point
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
	icons     iconSet
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
	r.icons, err = loadIconSet()
	if err != nil {
		return nil, err
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
	} else {
		r.renderDashboard(canvas, view, *view.Claude, activity)
	}
	renderTouchRipples(canvas, view.Touches, view.Now)
	return canvas
}

// renderTouchRipples draws a fading, expanding ring at each recent tap so
// the touchscreen visibly answers back even though nothing on this
// dashboard is otherwise interactive. Drawn last so it's never hidden
// under a card.
func renderTouchRipples(canvas *image.RGBA, points []touch.Point, now time.Time) {
	for _, point := range points {
		elapsed := now.Sub(point.At)
		if elapsed < 0 || elapsed > touchRippleLifetime {
			continue
		}
		progress := float64(elapsed) / float64(touchRippleLifetime)
		radius := 6 + int(46*progress)
		alpha := uint8(180 * (1 - progress))
		blendRing(canvas, point.X, point.Y, radius, max(2, 10-int(8*progress)), claudePeach, alpha)
	}
}

// contentSplit is where the open left panel ends and the framed Codex card
// begins. Both the rate-limit row above and the Codex card below start
// right after the header, full width, so neither side leaves a dead gap —
// codexTop is shared by both the dashboard and waiting screens so the card
// always starts exactly where the limit row ends.
const (
	contentSplit = contentLeft + 260
	codexTop     = 170
)

func (r *Renderer) renderDashboard(canvas *image.RGBA, view View, snapshot model.Snapshot, activity string) {
	modelName := snapshot.Model.DisplayName
	if modelName == "" {
		modelName = snapshot.Model.ID
	}
	sessionDetail := sessionName(snapshot) + "  •  " + modePrimary(snapshot)

	r.renderHeader(canvas, view, modelName, sessionDetail)
	r.renderRail(canvas, image.Rect(railLeft, sectionsTop, railRight, sectionsBottom), activity, &snapshot, view.Stats, view.Now)
	r.renderLimitsRow(canvas, snapshot, view.Now)
	r.renderClaudePanel(canvas, snapshot)
	r.codexCard(canvas, image.Rect(contentSplit+14, codexTop, contentRight, sectionsBottom), view.Codex)

	footer := fmt.Sprintf("AUTO  •  %d SESSION", max(1, view.SessionCount))
	if view.SessionCount != 1 {
		footer += "S"
	}
	if view.LoadError != nil {
		footer = "STATE WARNING  •  " + fitText(r.regular12, view.LoadError.Error(), 500)
	}
	r.text(canvas, r.bold13, claudeOrange, 21, footerBaseline, footer)
	r.textRight(canvas, r.regular12, textFaint, contentRight, footerBaseline, "CLAUDE PRIMARY  •  800×480")
}

// renderLimitsRow spans the full content width (not just the Claude panel)
// so the space above the Codex card — which deliberately starts lower,
// below this row — is never a dead gap.
func (r *Renderer) renderLimitsRow(canvas *image.RGBA, snapshot model.Snapshot, now time.Time) {
	fullWidth := contentRight - contentLeft
	halfWidth := (fullWidth - 14) / 2
	r.limitLine(canvas, contentLeft, 90, halfWidth, "5 HOUR", snapshot.RateLimits.FiveHour, snapshot.RateLimits, claudeOrange, now)
	r.limitLine(canvas, contentLeft+halfWidth+14, 90, halfWidth, "7 DAY", snapshot.RateLimits.SevenDay, snapshot.RateLimits, claudePeach, now)
}

// renderClaudePanel is deliberately card-less (open on the base background):
// the rail already frames "you", the Codex card already frames "the other
// tool", so this middle panel reads as this session's own numbers rather
// than another boxed widget competing for attention. Session/model/effort
// is written once, in the header, so this panel is context only, given the
// full height below the limit row down to the footer.
func (r *Renderer) renderClaudePanel(canvas *image.RGBA, snapshot model.Snapshot) {
	panelWidth := contentSplit - contentLeft

	r.text(canvas, r.bold13, textSecondary, contentLeft, 192, "CONTEXT WINDOW")
	r.contextBlock(canvas, contentLeft, 232, panelWidth, snapshot.Context, claudeOrange, r.bold44)

	chipWidth := (panelWidth - 14) / 2
	chipBounds := image.Rect(contentLeft, 294, contentLeft+chipWidth, 354)
	metricChip(canvas, r, chipBounds, "INPUT", tokenLabel(contextInput(snapshot.Context)), claudePeach)
	chipBounds = image.Rect(contentLeft+chipWidth+14, 294, contentSplit, 354)
	metricChip(canvas, r, chipBounds, "OUTPUT", tokenLabel(contextOutput(snapshot.Context)), purple)
}

func (r *Renderer) renderWaiting(canvas *image.RGBA, view View, activity string) {
	r.renderHeader(canvas, view, "", "")
	r.renderRail(canvas, image.Rect(railLeft, sectionsTop, railRight, sectionsBottom), activity, nil, view.Stats, view.Now)

	panelWidth := contentSplit - contentLeft
	center := contentLeft + panelWidth/2
	r.textCentered(canvas, r.bold22, textPrimary, center, 150, "WAITING")
	r.textCentered(canvas, r.regular13, textSecondary, center, 174, fitText(r.regular13, "for the next statusLine event", panelWidth-20))
	progress(canvas, image.Rect(contentLeft+20, 202, contentSplit-20, 212), animationPercent(view.Now), claudeOrange)
	r.textCentered(canvas, r.regular12, textFaint, center, 280, fitText(r.regular12, "start a session on the source machine", panelWidth-20))

	r.codexCard(canvas, image.Rect(contentSplit+14, codexTop, contentRight, sectionsBottom), view.Codex)

	r.text(canvas, r.bold13, claudeOrange, 21, footerBaseline, "CLAUDE PRIMARY  •  WAITING")
	r.textRight(canvas, r.regular12, textFaint, contentRight, footerBaseline, "FRAMEBUFFER 800×480")
}

// limitLine draws one open (no card) rate-limit indicator — label, big
// percentage, reset countdown, and a bar — stacked so the same compact
// shape works whether it's given half the Claude panel's width or more.
func (r *Renderer) limitLine(canvas *image.RGBA, x, top, width int, label string, window model.RateWindow, limits model.RateLimits, accent color.RGBA, now time.Time) {
	r.text(canvas, r.bold13, textSecondary, x, top, label+" LIMIT")
	if window.UsedPercentage == nil && limits.Unlimited != nil && *limits.Unlimited {
		r.text(canvas, r.bold22, green, x, top+30, "UNMETERED")
		progress(canvas, image.Rect(x, top+58, x+width, top+66), 100, green)
		return
	}
	pct := percentValue(window.UsedPercentage)
	r.text(canvas, r.bold22, textPrimary, x, top+30, percentLabel(window.UsedPercentage))
	r.text(canvas, r.regular12, textFaint, x, top+48, fitText(r.regular12, resetLabelShort(window.ResetsAt, now), width))
	progress(canvas, image.Rect(x, top+58, x+width, top+66), pct, thresholdColor(pct, accent))
}

// contextBlock draws percent+USED, the token fraction, and a bar with no
// card background, reused by both the Claude panel (bigger percentFace) and
// the Codex card (smaller one) so the two read as the same kind of number.
func (r *Renderer) contextBlock(canvas *image.RGBA, x, top, width int, context model.Context, accent color.RGBA, percentFace font.Face) {
	percentText := percentLabel(context.UsedPercentage)
	r.text(canvas, percentFace, textPrimary, x, top, percentText)
	usedX := x + font.MeasureString(percentFace, percentText).Ceil() + 10
	r.text(canvas, r.bold13, accent, usedX, top, "USED")
	r.text(canvas, r.regular12, textFaint, x, top+22, fitText(r.regular12, contextFraction(context), width))
	progress(canvas, image.Rect(x, top+34, x+width, top+42), percentValue(context.UsedPercentage), accent)
}

// renderHeader's sessionDetail is the one place session name/model/effort
// gets written — nowhere else on the dashboard repeats it.
func (r *Renderer) renderHeader(canvas *image.RGBA, view View, modelName, sessionDetail string) {
	r.drawLogoMark(canvas, 33, 36, 9)
	r.text(canvas, r.bold22, textPrimary, 54, 37, "CLAUDE")
	if modelName != "" {
		r.text(canvas, r.regular13, textSecondary, 54, 54, strings.ToUpper(fitText(r.regular13, modelName, 200)))
	}
	if sessionDetail != "" {
		r.text(canvas, r.regular12, textFaint, 54, 71, fitText(r.regular12, sessionDetail, 480))
	}

	if view.Claude != nil {
		age := view.Now.Sub(view.Claude.CapturedAt)
		if age < 0 {
			age = 0
		}
		liveText, liveColor := "LIVE", green
		if age > view.StaleAfter {
			liveText, liveColor = "STALE", red
		}
		// Size the pill to the label instead of a fixed width: "STALE" is
		// wider than "LIVE" and was overflowing its pill into the clock.
		clockZoneLeft := contentRight - 80
		textWidth := font.MeasureString(r.bold13, liveText).Ceil()
		pillWidth := textWidth + 34
		pillLeft := clockZoneLeft - 20 - pillWidth
		fillRounded(canvas, image.Rect(pillLeft, 18, pillLeft+pillWidth, 46), 14, withAlpha(liveColor, 36))
		fillCircle(canvas, pillLeft+15, 32, 4, liveColor)
		r.text(canvas, r.bold13, liveColor, pillLeft+26, 37, liveText)
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
	r.piHealthBox(canvas, image.Rect(bounds.Min.X+22, healthTop+14, bounds.Max.X-22, healthTop+70), stats)
}

// piHealthBox is one bordered row with three columns (CPU, MEM, GPU),
// label above value, divided by thin vertical rules.
func (r *Renderer) piHealthBox(canvas *image.RGBA, bounds image.Rectangle, stats systeminfo.Stats) {
	fillRounded(canvas, bounds, 12, cardStrong)
	segmentWidth := bounds.Dx() / 3
	labels := [3]string{"CPU", "MEM", "GPU"}
	values := [3]string{
		floatLabel(stats.CPUPercent, "%.0f%%"),
		memoryLabel(stats.MemoryUsedBytes, stats.MemoryTotalBytes),
		// GPU load has no reliable source on this hardware: the dashboard
		// writes straight to /dev/fb0 and never touches the 3D pipeline,
		// and the v3d core runs at a fixed clock regardless of load, so
		// there's nothing meaningful to sample — stays "--" like any
		// other unavailable value in this dashboard.
		"--",
	}
	for index := range labels {
		centerX := bounds.Min.X + segmentWidth*index + segmentWidth/2
		if index > 0 {
			x := bounds.Min.X + segmentWidth*index
			fillRounded(canvas, image.Rect(x, bounds.Min.Y+8, x+1, bounds.Max.Y-8), 0, trackColor)
		}
		r.textCentered(canvas, r.regular12, textFaint, centerX, bounds.Min.Y+22, labels[index])
		r.textCentered(canvas, r.bold13, textPrimary, centerX, bounds.Min.Y+44, fitText(r.bold13, values[index], segmentWidth-6))
	}
}

// codexCard is the one framed card on screen: it deliberately looks like a
// boxed "other tool" widget next to the Claude panel's open background. It
// stays confined to the middle+bottom of the content area — session/model
// and context only, no rate-limit rows of its own.
// codexCard intentionally omits session name/id — Codex is "the other
// tool" here, so only what model/reasoning it's running and how much
// context it's using matters on this dashboard, not which thread.
func (r *Renderer) codexCard(canvas *image.RGBA, bounds image.Rectangle, snapshot *model.Snapshot) {
	card(canvas, bounds, 19)
	r.text(canvas, r.bold13, green, bounds.Min.X+18, bounds.Min.Y+24, "CODEX")
	if snapshot == nil {
		r.text(canvas, r.bold16, textPrimary, bounds.Min.X+18, bounds.Min.Y+52, "NO SESSION")
		r.text(canvas, r.regular12, textFaint, bounds.Min.X+18, bounds.Min.Y+76, "context unavailable")
		return
	}

	innerWidth := bounds.Dx() - 36
	r.text(canvas, r.bold18, textPrimary, bounds.Min.X+18, bounds.Min.Y+52, fitText(r.bold18, strings.ToUpper(modelLabel(*snapshot)), innerWidth))
	r.text(canvas, r.regular12, textSecondary, bounds.Min.X+18, bounds.Min.Y+74, modePrimary(*snapshot))

	r.text(canvas, r.bold13, textSecondary, bounds.Min.X+18, bounds.Min.Y+106, "CONTEXT")
	r.contextBlock(canvas, bounds.Min.X+18, bounds.Min.Y+134, innerWidth, snapshot.Context, green, r.bold30)
}

func metricChip(canvas *image.RGBA, r *Renderer, bounds image.Rectangle, label, value string, accent color.RGBA) {
	fillRounded(canvas, bounds, 13, cardStrong)
	r.text(canvas, r.bold13, accent, bounds.Min.X+14, bounds.Min.Y+24, label)
	r.text(canvas, r.bold18, textPrimary, bounds.Min.X+14, bounds.Min.Y+50, fitText(r.bold18, value, bounds.Dx()-24))
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

// blendRing alpha-blends a ring (annulus) of the given color and alpha over
// whatever is already on canvas, unlike withAlpha which only blends toward
// a fixed card background. Used for the touch ripple, which can appear over
// the rail, the open panel, or the Codex card.
func blendRing(canvas *image.RGBA, centerX, centerY, radius, thickness int, tint color.RGBA, alpha uint8) {
	if alpha == 0 || radius <= 0 {
		return
	}
	inner := radius - thickness
	bounds := image.Rect(centerX-radius, centerY-radius, centerX+radius+1, centerY+radius+1).Intersect(canvas.Bounds())
	factor := float64(alpha) / 255
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		dy := y - centerY
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dx := x - centerX
			distanceSquared := dx*dx + dy*dy
			if distanceSquared > radius*radius || (inner > 0 && distanceSquared < inner*inner) {
				continue
			}
			existing := canvas.RGBAAt(x, y)
			canvas.SetRGBA(x, y, rgb(
				uint8(float64(tint.R)*factor+float64(existing.R)*(1-factor)),
				uint8(float64(tint.G)*factor+float64(existing.G)*(1-factor)),
				uint8(float64(tint.B)*factor+float64(existing.B)*(1-factor)),
			))
		}
	}
}

// activityVisual is the animation "voice" around each context-specific
// Clawd SVG: working bounces, while idle and approval breathe at different
// speeds and colors.
type activityVisual struct {
	period  time.Duration
	halo    color.RGBA
	bounce  bool
	breathe bool
}

func visualForActivity(state string) activityVisual {
	switch state {
	case model.ActivityWorking:
		return activityVisual{period: 900 * time.Millisecond, halo: rgb(58, 40, 32), bounce: true}
	case model.ActivityWaitingApproval:
		return activityVisual{period: 1100 * time.Millisecond, halo: rgb(58, 49, 24), breathe: true}
	default:
		return activityVisual{period: 2400 * time.Millisecond, halo: rgb(42, 34, 30), breathe: true}
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
	periodMS := max(int64(1), visual.period.Milliseconds())
	phase := float64(now.UnixMilli()%periodMS) / float64(periodMS) * 2 * math.Pi
	haloRadius := radius + 10
	halo := visual.halo
	if visual.breathe {
		haloRadius += int(2 * (math.Sin(phase) + 1))
		alpha := uint8(58 + 24*((math.Sin(phase)+1)/2))
		tint := claudePeach
		if state == model.ActivityWaitingApproval {
			tint = yellow
		}
		halo = withAlpha(tint, alpha)
	}
	fillCircle(canvas, centerX, centerY, haloRadius, halo)

	icon := r.icons.idle
	yOffset := 0
	switch state {
	case model.ActivityWorking:
		icon = r.icons.working
		if visual.bounce {
			yOffset = int(4 * math.Sin(phase))
		}
	case model.ActivityWaitingApproval:
		icon = r.icons.waitingApproval
	}
	drawIconCentered(canvas, icon, centerX, centerY+yOffset)
}

// drawLogoMark is the current Claude starburst, kept static so the rail owns
// all activity motion.
func (r *Renderer) drawLogoMark(canvas *image.RGBA, centerX, centerY, _ int) {
	drawIconCentered(canvas, r.icons.logo, centerX, centerY)
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

// resetLabelShort formats a rate-limit reset countdown for the compact
// stacked limitLine layout, where width is at a premium.
func resetLabelShort(epoch *int64, now time.Time) string {
	if epoch == nil || *epoch <= 0 {
		return "reset unavailable"
	}
	remaining := time.Unix(*epoch, 0).Sub(now)
	if remaining <= 0 {
		return "reset due"
	}
	return durationLabel(remaining) + " left"
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

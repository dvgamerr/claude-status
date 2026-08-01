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

	r.renderHeader(canvas, view, activity, modelName, sessionDetail)
	r.renderRail(canvas, image.Rect(railLeft, sectionsTop, railRight, sectionsBottom), activity, &snapshot, view.Stats, view.Now)
	r.renderLimitsRow(canvas, snapshot)
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
func (r *Renderer) renderLimitsRow(canvas *image.RGBA, snapshot model.Snapshot) {
	fullWidth := contentRight - contentLeft
	halfWidth := (fullWidth - 14) / 2
	r.limitLine(canvas, contentLeft, 90, halfWidth, "5 HOUR", snapshot.RateLimits.FiveHour, snapshot.RateLimits, claudeOrange)
	r.limitLine(canvas, contentLeft+halfWidth+14, 90, halfWidth, "7 DAY", snapshot.RateLimits.SevenDay, snapshot.RateLimits, claudePeach)
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
	r.renderHeader(canvas, view, activity, "", "")
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

// limitLine draws one open (no card) rate-limit indicator in exactly three
// lines — label, big percentage, bar — stacked so the same compact shape
// works whether it's given half the Claude panel's width or more.
func (r *Renderer) limitLine(canvas *image.RGBA, x, top, width int, label string, window model.RateWindow, limits model.RateLimits, accent color.RGBA) {
	r.text(canvas, r.bold13, textSecondary, x, top, label+" LIMIT")
	if window.UsedPercentage == nil && limits.Unlimited != nil && *limits.Unlimited {
		r.text(canvas, r.bold22, green, x, top+30, "UNMETERED")
		progress(canvas, image.Rect(x, top+40, x+width, top+48), 100, green)
		return
	}
	pct := percentValue(window.UsedPercentage)
	r.text(canvas, r.bold22, textPrimary, x, top+30, percentLabel(window.UsedPercentage))
	progress(canvas, image.Rect(x, top+40, x+width, top+48), pct, thresholdColor(pct, accent))
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
func (r *Renderer) renderHeader(canvas *image.RGBA, view View, activityState, modelName, sessionDetail string) {
	r.drawMascot(canvas, 33, 36, 9, view.Now, activityState)
	r.text(canvas, r.bold22, textPrimary, 54, 37, "CLAUDE")
	subtitle := "PRIMARY STATUS"
	if modelName != "" {
		subtitle += "  •  " + strings.ToUpper(fitText(r.regular13, modelName, 200))
	}
	r.text(canvas, r.regular13, textSecondary, 54, 54, subtitle)
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

// codexCard is the one framed card on screen: it deliberately looks like a
// boxed "other tool" widget next to the Claude panel's open background. It
// stays confined to the middle+bottom of the content area — session/model
// and context only, no rate-limit rows of its own.
func (r *Renderer) codexCard(canvas *image.RGBA, bounds image.Rectangle, snapshot *model.Snapshot) {
	card(canvas, bounds, 19)
	r.text(canvas, r.bold13, green, bounds.Min.X+18, bounds.Min.Y+24, "CODEX")
	r.textRight(canvas, r.regular12, textFaint, bounds.Max.X-17, bounds.Min.Y+24, "SESSION")
	if snapshot == nil {
		r.text(canvas, r.bold16, textPrimary, bounds.Min.X+18, bounds.Min.Y+52, "NO SESSION")
		r.text(canvas, r.regular12, textFaint, bounds.Min.X+18, bounds.Min.Y+76, "context unavailable")
		return
	}

	innerWidth := bounds.Dx() - 36
	r.text(canvas, r.bold18, textPrimary, bounds.Min.X+18, bounds.Min.Y+52, fitText(r.bold18, sessionName(*snapshot), innerWidth))
	r.text(canvas, r.regular12, textSecondary, bounds.Min.X+18, bounds.Min.Y+74, fitText(r.regular12, strings.ToUpper(modelLabel(*snapshot)), innerWidth))

	r.text(canvas, r.bold13, textSecondary, bounds.Min.X+18, bounds.Min.Y+106, "CONTEXT")
	r.contextBlock(canvas, bounds.Min.X+18, bounds.Min.Y+134, innerWidth, snapshot.Context, green, r.bold30)

	chipWidth := (innerWidth - 14) / 2
	chipTop := bounds.Min.Y + 196
	metricChip(canvas, r, image.Rect(bounds.Min.X+18, chipTop, bounds.Min.X+18+chipWidth, chipTop+60), "INPUT", tokenLabel(contextInput(snapshot.Context)), claudePeach)
	metricChip(canvas, r, image.Rect(bounds.Max.X-18-chipWidth, chipTop, bounds.Max.X-18, chipTop+60), "OUTPUT", tokenLabel(contextOutput(snapshot.Context)), purple)
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

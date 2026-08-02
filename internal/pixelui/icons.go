package pixelui

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/draw"
	"io/fs"
	"sort"
	"time"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
)

const railIconSize = 120

// The SVGs stay as the source assets and are rasterized once when the
// renderer starts. That keeps the framebuffer render loop allocation-free
// while preserving one scalable source for both the 30 px header mark and
// the 120 px activity artwork.
//
//go:embed assets/anthropic.svg
var anthropicLogoSVG []byte

//go:embed assets/clawd-sleeping.svg
var clawdSleepingSVG []byte

//go:embed assets/clawd-sleeping-2.svg
var clawdSleeping2SVG []byte

//go:embed assets/clawd-exclamation-mark.svg
var clawdExclamationMarkSVG []byte

//go:embed assets/clawd-exclamation-mark-2.svg
var clawdExclamationMark2SVG []byte

// gifFrameDuration is how long each frame of the five sequences below stays
// on screen. They were traced from every 5th frame of a ~16.7fps (60ms)
// source GIF, so 5*60ms reproduces the original's motion at the same speed.
const gifFrameDuration = 300 * time.Millisecond

// These five states were traced frame-by-frame from real reference
// animations (see internal/pixelui/assets/README.md) rather than given a
// single pose plus procedural motion like the original three — each
// directory glob below embeds one numbered SVG per frame of that GIF's
// actual motion (thought bubble rising, hands typing, hammer swinging,
// head bobbing to music, balls arcing), in playback order.
//
//go:embed assets/clawd-thinking-*.svg
var clawdThinkingFrames embed.FS

//go:embed assets/clawd-typing-*.svg
var clawdTypingFrames embed.FS

//go:embed assets/clawd-building-*.svg
var clawdBuildingFrames embed.FS

//go:embed assets/clawd-headphones-groove-*.svg
var clawdHeadphonesGrooveFrames embed.FS

//go:embed assets/clawd-juggling-*.svg
var clawdJugglingFrames embed.FS

// Each activity keeps two rasterized SVG frames — not just the whole-icon
// bob/scale applied in render.go, but an actual second pose (typing hands,
// drifted Zzz, a pulsing alert dot) baked into the artwork itself, so the
// mascot's parts move independently instead of the whole icon rigidly
// sliding as one block.
type iconSet struct {
	logo            *image.RGBA
	idle            [2]*image.RGBA
	waitingApproval [2]*image.RGBA
	// thinking/typing/building/subagentOne/subagentMany are full traced GIF
	// sequences (see gifFrameDuration) rather than a 2-frame alternation.
	thinking     []*image.RGBA
	typing       []*image.RGBA
	building     []*image.RGBA
	subagentOne  []*image.RGBA
	subagentMany []*image.RGBA
}

func loadIconSet() (iconSet, error) {
	load := func(name string, source []byte, width, height int) (*image.RGBA, error) {
		icon, err := rasterizeSVG(source, width, height)
		if err != nil {
			return nil, fmt.Errorf("render %s SVG: %w", name, err)
		}
		return icon, nil
	}

	var set iconSet
	var err error
	if set.logo, err = load("Anthropic logo", anthropicLogoSVG, 30, 30); err != nil {
		return iconSet{}, err
	}
	if set.idle[0], err = load("Clawd Sleeping", clawdSleepingSVG, railIconSize, railIconSize); err != nil {
		return iconSet{}, err
	}
	if set.idle[1], err = load("Clawd Sleeping frame 2", clawdSleeping2SVG, railIconSize, railIconSize); err != nil {
		return iconSet{}, err
	}
	if set.waitingApproval[0], err = load("Clawd Exclamation Mark", clawdExclamationMarkSVG, railIconSize, railIconSize); err != nil {
		return iconSet{}, err
	}
	if set.waitingApproval[1], err = load("Clawd Exclamation Mark frame 2", clawdExclamationMark2SVG, railIconSize, railIconSize); err != nil {
		return iconSet{}, err
	}

	sequences := []struct {
		name        string
		fsys        embed.FS
		destination *[]*image.RGBA
	}{
		{"Clawd Thinking", clawdThinkingFrames, &set.thinking},
		{"Clawd Typing", clawdTypingFrames, &set.typing},
		{"Clawd Building", clawdBuildingFrames, &set.building},
		{"Clawd Headphones Groove", clawdHeadphonesGrooveFrames, &set.subagentOne},
		{"Clawd Juggling", clawdJugglingFrames, &set.subagentMany},
	}
	for _, sequence := range sequences {
		frames, err := loadFrameSequence(sequence.name, sequence.fsys, railIconSize, railIconSize)
		if err != nil {
			return iconSet{}, err
		}
		*sequence.destination = frames
	}
	return set, nil
}

// loadFrameSequence rasterizes every SVG embedded via a "assets/clawd-X-*.svg"
// glob, in filename order — frame numbers are zero-padded (e.g. "-01.svg",
// "-10.svg") specifically so this lexical sort is also playback order.
func loadFrameSequence(name string, fsys embed.FS, width, height int) ([]*image.RGBA, error) {
	entries, err := fs.Glob(fsys, "assets/*.svg")
	if err != nil {
		return nil, fmt.Errorf("glob %s frames: %w", name, err)
	}
	sort.Strings(entries)
	frames := make([]*image.RGBA, 0, len(entries))
	for _, entry := range entries {
		source, err := fsys.ReadFile(entry)
		if err != nil {
			return nil, fmt.Errorf("read %s frame %q: %w", name, entry, err)
		}
		icon, err := rasterizeSVG(source, width, height)
		if err != nil {
			return nil, fmt.Errorf("render %s frame %q: %w", name, entry, err)
		}
		frames = append(frames, icon)
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames found for %s", name)
	}
	return frames, nil
}

func rasterizeSVG(source []byte, width, height int) (*image.RGBA, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	icon.SetTarget(0, 0, float64(width), float64(height))
	destination := image.NewRGBA(image.Rect(0, 0, width, height))
	scanner := rasterx.NewScannerGV(width, height, destination, destination.Bounds())
	icon.Draw(rasterx.NewDasher(width, height, scanner), 1)
	return destination, nil
}

func drawIconCentered(canvas *image.RGBA, icon *image.RGBA, centerX, centerY int) {
	drawIconScaledCentered(canvas, icon, centerX, centerY, icon.Bounds().Dx(), icon.Bounds().Dy())
}

func drawIconScaledCentered(canvas *image.RGBA, icon *image.RGBA, centerX, centerY, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	destination := iconDestination(centerX, centerY, width, height)
	bounds := icon.Bounds()
	if width == bounds.Dx() && height == bounds.Dy() {
		draw.Draw(canvas, destination, icon, bounds.Min, draw.Over)
		return
	}
	// Keep the deliberately blocky Clawd artwork crisp while it breathes.
	// Bilinear scaling makes the one-pixel SVG edges look blurry on RGB565.
	xdraw.NearestNeighbor.Scale(canvas, destination, icon, bounds, draw.Over, nil)
}

func iconDestination(centerX, centerY, width, height int) image.Rectangle {
	left := centerX - width/2
	top := centerY - height/2
	return image.Rect(left, top, left+width, top+height)
}

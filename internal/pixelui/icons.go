package pixelui

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/draw"

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

//go:embed assets/clawd-coding.svg
var clawdCodingSVG []byte

//go:embed assets/clawd-coding-2.svg
var clawdCoding2SVG []byte

//go:embed assets/clawd-sleeping.svg
var clawdSleepingSVG []byte

//go:embed assets/clawd-sleeping-2.svg
var clawdSleeping2SVG []byte

//go:embed assets/clawd-exclamation-mark.svg
var clawdExclamationMarkSVG []byte

//go:embed assets/clawd-exclamation-mark-2.svg
var clawdExclamationMark2SVG []byte

// Each activity keeps two rasterized SVG frames — not just the whole-icon
// bob/scale applied in render.go, but an actual second pose (typing hands,
// drifted Zzz, a pulsing alert dot) baked into the artwork itself, so the
// mascot's parts move independently instead of the whole icon rigidly
// sliding as one block.
type iconSet struct {
	logo            *image.RGBA
	working         [2]*image.RGBA
	idle            [2]*image.RGBA
	waitingApproval [2]*image.RGBA
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
	if set.working[0], err = load("Clawd Coding", clawdCodingSVG, railIconSize, railIconSize); err != nil {
		return iconSet{}, err
	}
	if set.working[1], err = load("Clawd Coding frame 2", clawdCoding2SVG, railIconSize, railIconSize); err != nil {
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
	return set, nil
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

package pixelui

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/draw"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
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

//go:embed assets/clawd-sleeping.svg
var clawdSleepingSVG []byte

//go:embed assets/clawd-exclamation-mark.svg
var clawdExclamationMarkSVG []byte

type iconSet struct {
	logo            *image.RGBA
	working         *image.RGBA
	idle            *image.RGBA
	waitingApproval *image.RGBA
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
	if set.working, err = load("Clawd Coding", clawdCodingSVG, railIconSize, railIconSize); err != nil {
		return iconSet{}, err
	}
	if set.idle, err = load("Clawd Sleeping", clawdSleepingSVG, railIconSize, railIconSize); err != nil {
		return iconSet{}, err
	}
	if set.waitingApproval, err = load("Clawd Exclamation Mark", clawdExclamationMarkSVG, railIconSize, railIconSize); err != nil {
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
	bounds := icon.Bounds()
	destination := image.Rect(
		centerX-bounds.Dx()/2,
		centerY-bounds.Dy()/2,
		centerX-bounds.Dx()/2+bounds.Dx(),
		centerY-bounds.Dy()/2+bounds.Dy(),
	)
	draw.Draw(canvas, destination, icon, bounds.Min, draw.Over)
}

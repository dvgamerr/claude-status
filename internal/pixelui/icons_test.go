package pixelui

import (
	"image"
	"testing"
)

// TestRasterizeSVGError feeds rasterizeSVG a deliberately malformed SVG byte
// slice constructed in-test (not one of the embedded assets) so the error
// path is exercised without corrupting any real asset. The stroke-width
// attribute is valid XML but not a parsable number, so oksvg.ReadIconStream
// fails while parsing the root element's style attributes.
func TestRasterizeSVGError(t *testing.T) {
	bad := []byte(`<svg xmlns="http://www.w3.org/2000/svg" stroke-width="not-a-number"></svg>`)
	if _, err := rasterizeSVG(bad, 10, 10); err == nil {
		t.Fatal("rasterizeSVG accepted an SVG with an unparsable style attribute")
	}
}

// TestSmokeTestRigEvaluateError covers smokeTestRig's first error branch:
// svganim.Evaluate failing on malformed/truncated XML before rasterization
// is ever attempted.
func TestSmokeTestRigEvaluateError(t *testing.T) {
	if err := smokeTestRig("broken rig", []byte("<svg><rect")); err == nil {
		t.Fatal("smokeTestRig accepted truncated XML")
	}
}

// TestSmokeTestRigRasterizeError covers smokeTestRig's second error branch:
// the source is valid, boring XML (so svganim.Evaluate succeeds and returns
// it basically unchanged) but oksvg then fails to rasterize it because of an
// unparsable style attribute.
func TestSmokeTestRigRasterizeError(t *testing.T) {
	bad := []byte(`<svg xmlns="http://www.w3.org/2000/svg" stroke-width="not-a-number"></svg>`)
	if err := smokeTestRig("broken rig", bad); err == nil {
		t.Fatal("smokeTestRig accepted a rig that evaluates fine but fails to rasterize")
	}
}

func TestDrawIconScaledCenteredEdgeCases(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for x := 0; x < 20; x++ {
		for y := 0; y < 20; y++ {
			icon.Set(x, y, red)
		}
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 100, 100))
	before := image.NewRGBA(canvas.Bounds())
	copy(before.Pix, canvas.Pix)

	drawIconScaledCentered(canvas, icon, 50, 50, 0, 20)
	drawIconScaledCentered(canvas, icon, 50, 50, 20, 0)
	drawIconScaledCentered(canvas, icon, 50, 50, -5, 20)
	if !sameRegion(canvas, before, canvas.Bounds()) {
		t.Fatal("drawIconScaledCentered drew despite a non-positive destination dimension")
	}

	// A destination size that differs from the icon's own bounds takes the
	// NearestNeighbor scaling path instead of the direct draw.Draw copy.
	drawIconScaledCentered(canvas, icon, 50, 50, 40, 40)
	if rgba(canvas.At(50, 50)) != red {
		t.Fatal("drawIconScaledCentered(scaled) did not paint the scaled icon")
	}
}

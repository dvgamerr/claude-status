//go:build !linux

package framebuffer

import (
	"image"
	"strings"
	"testing"
)

func TestUnsupportedPlatformScreen(t *testing.T) {
	screen, err := Open("/dev/fb0", "/dev/tty1")
	if screen != nil || err == nil || !strings.Contains(err.Error(), "only on Linux") {
		t.Fatalf("Open() screen=%v error=%v", screen, err)
	}
	empty := &Screen{}
	if empty.Size() != (image.Point{}) {
		t.Fatalf("Size() = %v", empty.Size())
	}
	if err := empty.Present(image.NewRGBA(image.Rect(0, 0, 1, 1))); err == nil {
		t.Fatal("Present() unexpectedly succeeded")
	}
	if err := empty.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

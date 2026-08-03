//go:build linux

package framebuffer

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestRGB565(t *testing.T) {
	if got := rgb565(255, 0, 0); got != 0xF800 {
		t.Fatalf("red RGB565 = %#04x", got)
	}
	if got := rgb565(0, 255, 0); got != 0x07E0 {
		t.Fatalf("green RGB565 = %#04x", got)
	}
	if got := rgb565(0, 0, 255); got != 0x001F {
		t.Fatalf("blue RGB565 = %#04x", got)
	}
}

func TestPresentConvertsFastAndGenericImagePaths(t *testing.T) {
	screen := &Screen{memory: make([]byte, 8), width: 2, height: 2, stride: 4}
	rgba := image.NewRGBA(image.Rect(0, 0, 2, 2))
	rgba.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	rgba.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	rgba.SetRGBA(0, 1, color.RGBA{B: 255, A: 255})
	if err := screen.Present(rgba); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x00, 0xF8, 0xE0, 0x07, 0x1F, 0x00, 0x00, 0x00}
	if string(screen.memory) != string(want) {
		t.Fatalf("fast RGB565 bytes = %v, want %v", screen.memory, want)
	}

	generic := image.NewNRGBA(image.Rect(10, 20, 12, 22))
	generic.Set(10, 20, color.White)
	if err := screen.Present(generic); err != nil {
		t.Fatal(err)
	}
	if screen.memory[0] != 0xFF || screen.memory[1] != 0xFF {
		t.Fatalf("generic RGB565 white = %#02x %#02x", screen.memory[0], screen.memory[1])
	}
}

func TestPresentRejectsInvalidFrames(t *testing.T) {
	screen := &Screen{memory: make([]byte, 8), width: 2, height: 2, stride: 4}
	if err := screen.Present(nil); err == nil {
		t.Fatal("Present accepted nil frame")
	}
	if err := screen.Present(image.NewRGBA(image.Rect(0, 0, 1, 1))); err == nil {
		t.Fatal("Present accepted wrong frame size")
	}
	screen.closed = true
	if err := screen.Present(image.NewRGBA(image.Rect(0, 0, 2, 2))); err == nil {
		t.Fatal("Present accepted frame after close")
	}
}

func TestReadFramebufferMetadata(t *testing.T) {
	dir := t.TempDir()
	pairPath := filepath.Join(dir, "virtual_size")
	intPath := filepath.Join(dir, "stride")
	if err := os.WriteFile(pairPath, []byte("800,480\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(intPath, []byte("1600\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	width, height, err := readPair(pairPath)
	if err != nil || width != 800 || height != 480 {
		t.Fatalf("readPair() = %d,%d,%v", width, height, err)
	}
	stride, err := readInt(intPath)
	if err != nil || stride != 1600 {
		t.Fatalf("readInt() = %d,%v", stride, err)
	}
}

func TestValidateGeometry(t *testing.T) {
	tests := []struct {
		name                  string
		width, height, stride int
		want                  int
		wantError             bool
	}{
		{"valid", 800, 480, 1600, 768000, false},
		{"zero width", 0, 480, 1600, 0, true},
		{"short stride", 800, 480, 1599, 0, true},
		{"overflow", 1, int(^uint(0) >> 1), 3, 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := validateGeometry(test.width, test.height, test.stride)
			if (err != nil) != test.wantError || got != test.want {
				t.Fatalf("validateGeometry() = %d, %v", got, err)
			}
		})
	}
}

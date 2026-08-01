//go:build linux

package framebuffer

import (
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

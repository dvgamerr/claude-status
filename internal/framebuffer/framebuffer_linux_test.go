//go:build linux

package framebuffer

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
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

func TestReadPairErrorBranches(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := readPair(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("readPair() on missing file: error = nil")
	}
	single := filepath.Join(dir, "single")
	if err := os.WriteFile(single, []byte("800\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readPair(single); err == nil || !strings.Contains(err.Error(), "invalid framebuffer size") {
		t.Fatalf("readPair() on single value: error = %v", err)
	}
	badWidth := filepath.Join(dir, "bad-width")
	if err := os.WriteFile(badWidth, []byte("x,480\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readPair(badWidth); err == nil || !strings.Contains(err.Error(), "parse framebuffer width") {
		t.Fatalf("readPair() on bad width: error = %v", err)
	}
	badHeight := filepath.Join(dir, "bad-height")
	if err := os.WriteFile(badHeight, []byte("800,y\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readPair(badHeight); err == nil || !strings.Contains(err.Error(), "parse framebuffer height") {
		t.Fatalf("readPair() on bad height: error = %v", err)
	}
}

func TestReadIntErrorBranches(t *testing.T) {
	dir := t.TempDir()
	if _, err := readInt(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("readInt() on missing file: error = nil")
	}
	bad := filepath.Join(dir, "bad")
	if err := os.WriteFile(bad, []byte("sixteen\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readInt(bad); err == nil {
		t.Fatal("readInt() on non-numeric content: error = nil")
	}
}

func writeSysfsFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeGraphicsDevice builds a /sys/class/graphics/<name>/{virtual_size,
// bits_per_pixel,stride} tree under a temp dir and points sysfsGraphicsRoot
// at it, so Open's metadata-reading branches can run against a real
// filesystem without touching the actual Pi hardware.
func fakeGraphicsDevice(t *testing.T, name string, virtualSize, bitsPerPixel, stride string) string {
	t.Helper()
	root := t.TempDir()
	dev := filepath.Join(root, name)
	if err := os.MkdirAll(dev, 0o755); err != nil {
		t.Fatal(err)
	}
	if virtualSize != "" {
		writeSysfsFile(t, dev, "virtual_size", virtualSize)
	}
	if bitsPerPixel != "" {
		writeSysfsFile(t, dev, "bits_per_pixel", bitsPerPixel)
	}
	if stride != "" {
		writeSysfsFile(t, dev, "stride", stride)
	}
	old := sysfsGraphicsRoot
	t.Cleanup(func() { sysfsGraphicsRoot = old })
	sysfsGraphicsRoot = root
	return root
}

func TestOpenMissingVirtualSize(t *testing.T) {
	fakeGraphicsDevice(t, "fb0", "", "16", "1600")
	if _, err := Open(filepath.Join(t.TempDir(), "fb0"), "/dev/null"); err == nil || !strings.Contains(err.Error(), "virtual_size") {
		t.Fatalf("Open() error = %v, want a virtual_size read error", err)
	}
}

func TestOpenNonNumericVirtualSizeWidth(t *testing.T) {
	fakeGraphicsDevice(t, "fb0", "x,480", "16", "1600")
	if _, err := Open(filepath.Join(t.TempDir(), "fb0"), "/dev/null"); err == nil || !strings.Contains(err.Error(), "parse framebuffer width") {
		t.Fatalf("Open() error = %v, want a width parse error", err)
	}
}

func TestOpenMissingBitsPerPixel(t *testing.T) {
	fakeGraphicsDevice(t, "fb0", "800,480", "", "1600")
	if _, err := Open(filepath.Join(t.TempDir(), "fb0"), "/dev/null"); err == nil || !strings.Contains(err.Error(), "bits_per_pixel") {
		t.Fatalf("Open() error = %v, want a bits_per_pixel read error", err)
	}
}

func TestOpenRejectsNon16Bpp(t *testing.T) {
	fakeGraphicsDevice(t, "fb0", "800,480", "32", "3200")
	_, err := Open(filepath.Join(t.TempDir(), "fb0"), "/dev/null")
	if err == nil || !strings.Contains(err.Error(), "RGB565") {
		t.Fatalf("Open() error = %v, want an RGB565-required error", err)
	}
}

func TestOpenMissingStride(t *testing.T) {
	fakeGraphicsDevice(t, "fb0", "800,480", "16", "")
	if _, err := Open(filepath.Join(t.TempDir(), "fb0"), "/dev/null"); err == nil || !strings.Contains(err.Error(), "stride") {
		t.Fatalf("Open() error = %v, want a stride read error", err)
	}
}

func TestOpenInvalidGeometry(t *testing.T) {
	// stride smaller than width*2 fails validateGeometry, not readInt.
	fakeGraphicsDevice(t, "fb0", "800,480", "16", "10")
	_, err := Open(filepath.Join(t.TempDir(), "fb0"), "/dev/null")
	if err == nil || !strings.Contains(err.Error(), "invalid framebuffer geometry") {
		t.Fatalf("Open() error = %v, want an invalid geometry error", err)
	}
}

func TestOpenFramebufferDeviceMissing(t *testing.T) {
	fakeGraphicsDevice(t, "fb0", "800,480", "16", "1600")
	missing := filepath.Join(t.TempDir(), "fb0")
	if _, err := Open(missing, "/dev/null"); err == nil || !strings.Contains(err.Error(), "open framebuffer") {
		t.Fatalf("Open() error = %v, want an open-framebuffer error", err)
	}
}

func TestOpenTTYDeviceMissing(t *testing.T) {
	fakeGraphicsDevice(t, "fb0", "800,480", "16", "1600")
	fbPath := filepath.Join(t.TempDir(), "fb0")
	if err := os.WriteFile(fbPath, make([]byte, 1600*480), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(fbPath, filepath.Join(t.TempDir(), "no-such-tty")); err == nil || !strings.Contains(err.Error(), "open graphics tty") {
		t.Fatalf("Open() error = %v, want an open-tty error", err)
	}
}

// TestOpenIoctlFailsOnNonTTY uses a plain regular file as the "tty": the
// real KDSETMODE ioctl genuinely fails with ENOTTY against it, exercising
// Open's ioctl-failure cleanup path without touching any real console.
func TestOpenIoctlFailsOnNonTTY(t *testing.T) {
	fakeGraphicsDevice(t, "fb0", "800,480", "16", "1600")
	fbPath := filepath.Join(t.TempDir(), "fb0")
	if err := os.WriteFile(fbPath, make([]byte, 1600*480), 0o644); err != nil {
		t.Fatal(err)
	}
	ttyPath := filepath.Join(t.TempDir(), "faketty")
	if err := os.WriteFile(ttyPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(fbPath, ttyPath); err == nil || !strings.Contains(err.Error(), "graphics mode") {
		t.Fatalf("Open() error = %v, want a graphics-mode switch error", err)
	}
}

func TestScreenSize(t *testing.T) {
	s := &Screen{width: 800, height: 480}
	if got := s.Size(); got != image.Pt(800, 480) {
		t.Fatalf("Size() = %v, want 800x480", got)
	}
}

func TestScreenCloseIsIdempotent(t *testing.T) {
	s := &Screen{closed: true}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() on already-closed screen error = %v", err)
	}
}

func TestScreenCloseUnmapsRealMapping(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fb")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	const size = 4096
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
	memory, err := unix.Mmap(int(file.Fd()), 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		t.Fatalf("test setup: Mmap() error = %v", err)
	}

	ttyFile, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer ttyFile.Close()

	s := &Screen{framebuffer: file, tty: ttyFile, memory: memory, width: 1, height: 1, stride: 2}
	// The real ioctl restoring text mode fails against a regular file
	// (ENOTTY) — a genuine error, not a fake one — so Close() still
	// returns a non-nil joined error even though Munmap/Close succeed.
	if err := s.Close(); err == nil || !strings.Contains(err.Error(), "restore tty text mode") {
		t.Fatalf("Close() error = %v, want a tty-text-mode restore error", err)
	}
}

func TestScreenCloseReportsAlreadyClosedFileHandles(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fb")
	if err != nil {
		t.Fatal(err)
	}
	ttyFile, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	// Close both real handles ahead of time so Screen.Close's own Close
	// calls on them deterministically fail with "file already closed",
	// covering those error branches without touching any device file.
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ttyFile.Close(); err != nil {
		t.Fatal(err)
	}

	s := &Screen{framebuffer: file, tty: ttyFile, memory: []byte{}, width: 0, height: 0, stride: 0}
	err = s.Close()
	if err == nil {
		t.Fatal("Close() error = nil, want errors from the already-closed handles")
	}
	if !strings.Contains(err.Error(), "close tty") || !strings.Contains(err.Error(), "close framebuffer") {
		t.Fatalf("Close() error = %v, want it to mention both close failures", err)
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

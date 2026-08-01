//go:build linux

package framebuffer

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	kdSetMode  = 0x4B3A
	kdText     = 0x00
	kdGraphics = 0x01
)

type Screen struct {
	framebuffer *os.File
	tty         *os.File
	memory      []byte
	width       int
	height      int
	stride      int
	closed      bool
	mutex       sync.Mutex
}

func Open(framebufferPath, ttyPath string) (*Screen, error) {
	name := filepath.Base(filepath.Clean(framebufferPath))
	sysfs := filepath.Join("/sys/class/graphics", name)
	width, height, err := readPair(filepath.Join(sysfs, "virtual_size"))
	if err != nil {
		return nil, err
	}
	bpp, err := readInt(filepath.Join(sysfs, "bits_per_pixel"))
	if err != nil {
		return nil, err
	}
	if bpp != 16 {
		return nil, fmt.Errorf("framebuffer %s uses %d bpp; RGB565 (16 bpp) is required", framebufferPath, bpp)
	}
	stride, err := readInt(filepath.Join(sysfs, "stride"))
	if err != nil {
		return nil, err
	}
	framebufferFile, err := os.OpenFile(framebufferPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open framebuffer %s: %w", framebufferPath, err)
	}
	ttyFile, err := os.OpenFile(ttyPath, os.O_RDWR, 0)
	if err != nil {
		framebufferFile.Close()
		return nil, fmt.Errorf("open graphics tty %s: %w", ttyPath, err)
	}
	if err := unix.IoctlSetInt(int(ttyFile.Fd()), kdSetMode, kdGraphics); err != nil {
		ttyFile.Close()
		framebufferFile.Close()
		return nil, fmt.Errorf("switch %s to graphics mode: %w", ttyPath, err)
	}
	memory, err := unix.Mmap(int(framebufferFile.Fd()), 0, stride*height, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		_ = unix.IoctlSetInt(int(ttyFile.Fd()), kdSetMode, kdText)
		ttyFile.Close()
		framebufferFile.Close()
		return nil, fmt.Errorf("map framebuffer %s: %w", framebufferPath, err)
	}
	return &Screen{framebuffer: framebufferFile, tty: ttyFile, memory: memory, width: width, height: height, stride: stride}, nil
}

func (s *Screen) Size() image.Point { return image.Pt(s.width, s.height) }

func (s *Screen) Present(source image.Image) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed {
		return errors.New("framebuffer is closed")
	}
	if source.Bounds().Dx() != s.width || source.Bounds().Dy() != s.height {
		return fmt.Errorf("frame is %dx%d; framebuffer is %dx%d", source.Bounds().Dx(), source.Bounds().Dy(), s.width, s.height)
	}
	if rgba, ok := source.(*image.RGBA); ok && rgba.Rect.Min == (image.Point{}) {
		for y := 0; y < s.height; y++ {
			sourceRow := rgba.Pix[y*rgba.Stride:]
			destinationRow := s.memory[y*s.stride:]
			for x := 0; x < s.width; x++ {
				offset := x * 4
				pixel := rgb565(sourceRow[offset], sourceRow[offset+1], sourceRow[offset+2])
				destinationRow[x*2] = byte(pixel)
				destinationRow[x*2+1] = byte(pixel >> 8)
			}
		}
		return nil
	}
	for y := 0; y < s.height; y++ {
		row := s.memory[y*s.stride:]
		for x := 0; x < s.width; x++ {
			red, green, blue, _ := source.At(source.Bounds().Min.X+x, source.Bounds().Min.Y+y).RGBA()
			pixel := rgb565(uint8(red>>8), uint8(green>>8), uint8(blue>>8))
			row[x*2] = byte(pixel)
			row[x*2+1] = byte(pixel >> 8)
		}
	}
	return nil
}

func (s *Screen) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	clear(s.memory)
	var result error
	if err := unix.Munmap(s.memory); err != nil {
		result = fmt.Errorf("unmap framebuffer: %w", err)
	}
	if err := unix.IoctlSetInt(int(s.tty.Fd()), kdSetMode, kdText); err != nil && result == nil {
		result = fmt.Errorf("restore tty text mode: %w", err)
	}
	if err := s.tty.Close(); err != nil && result == nil {
		result = err
	}
	if err := s.framebuffer.Close(); err != nil && result == nil {
		result = err
	}
	return result
}

func rgb565(red, green, blue uint8) uint16 {
	return uint16(red>>3)<<11 | uint16(green>>2)<<5 | uint16(blue>>3)
}

func readPair(path string) (int, int, error) {
	value, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, fmt.Errorf("read %s: %w", path, err)
	}
	parts := strings.Split(strings.TrimSpace(string(value)), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid framebuffer size %q", strings.TrimSpace(string(value)))
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse framebuffer width: %w", err)
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse framebuffer height: %w", err)
	}
	return width, height, nil
}

func readInt(path string) (int, error) {
	value, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(string(value)))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	return parsed, nil
}

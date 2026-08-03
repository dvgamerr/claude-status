//go:build !linux

// Package framebuffer presents RGB565 frames on Linux framebuffer devices.
package framebuffer

import (
	"errors"
	"image"
)

// Screen is the unsupported-platform placeholder for a native framebuffer.
type Screen struct{}

// Open reports that native framebuffer output is Linux-only.
func Open(string, string) (*Screen, error) {
	return nil, errors.New("native framebuffer dashboard is available only on Linux")
}

// Size returns an empty size on unsupported platforms.
func (*Screen) Size() image.Point { return image.Point{} }

// Present reports that native framebuffer output is Linux-only.
func (*Screen) Present(image.Image) error {
	return errors.New("native framebuffer dashboard is available only on Linux")
}

// Close is a no-op for an unsupported placeholder screen.
func (*Screen) Close() error { return nil }

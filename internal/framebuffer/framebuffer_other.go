//go:build !linux

package framebuffer

import (
	"errors"
	"image"
)

type Screen struct{}

func Open(string, string) (*Screen, error) {
	return nil, errors.New("native framebuffer dashboard is available only on Linux")
}

func (*Screen) Size() image.Point { return image.Point{} }
func (*Screen) Present(image.Image) error {
	return errors.New("native framebuffer dashboard is available only on Linux")
}
func (*Screen) Close() error { return nil }

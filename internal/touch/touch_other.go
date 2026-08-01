//go:build !linux

package touch

import (
	"context"
	"errors"
)

// Watch is unavailable outside Linux; the gfx dashboard only runs on the Pi,
// so callers on other platforms should treat this as "no touch input"
// rather than a fatal error.
func Watch(context.Context, string) (<-chan Point, error) {
	return nil, errors.New("touch input is available only on Linux")
}

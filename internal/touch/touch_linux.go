//go:build linux

package touch

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// input_event on 64-bit Linux (arm64, amd64) is 24 bytes: an 8-byte tv_sec,
// an 8-byte tv_usec, a 2-byte type, a 2-byte code, and a 4-byte value.
// Confirmed against a raw capture from the Pi's touch controller
// ("10-0038 generic ft5x06") rather than assumed from headers.
const eventSize = 24

const (
	evKey = 1
	evAbs = 3
	evSyn = 0

	synReport = 0

	absMTPositionX = 0x35
	absMTPositionY = 0x36

	btnTouch = 0x14a
)

// Watch opens a Linux evdev device (typically /dev/input/eventN for the
// touch controller) and emits one Point per finger-down contact. Position
// is read from the ABS_MT_POSITION_X/Y fields — confirmed by capture to
// already be scaled to the panel's 800x480 pixels — and a point is emitted
// only once a full input frame (ending in SYN_REPORT) confirms BTN_TOUCH
// went down, so ordering of fields within one contact report can't produce
// a stale or half-updated coordinate.
//
// The returned channel is closed when ctx is done or the device read
// fails; callers should treat a closed channel as "no more touch input"
// rather than a fatal error — the dashboard must keep running without it.
func Watch(ctx context.Context, devicePath string) (<-chan Point, error) {
	if ctx == nil {
		return nil, fmt.Errorf("watch touch device: context is nil")
	}
	device, err := os.Open(devicePath)
	if err != nil {
		return nil, fmt.Errorf("open touch device %s: %w", devicePath, err)
	}

	points := make(chan Point, 8)
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = device.Close()
		case <-done:
		}
	}()

	go func() {
		defer close(done)
		defer close(points)
		defer func() { _ = device.Close() }()

		var buffer [eventSize]byte
		var decoder eventDecoder
		for {
			if _, err := io.ReadFull(device, buffer[:]); err != nil {
				return
			}
			point, emit := decoder.decode(buffer[:], time.Now())
			if !emit {
				continue
			}
			select {
			case points <- point:
			case <-ctx.Done():
				return
			default:
				// A slow consumer drops the point; a missed ripple
				// is harmless and must never block reading input.
			}
		}
	}()
	return points, nil
}

type eventDecoder struct {
	x, y          int
	pressed       bool
	pendingRipple bool
}

func (decoder *eventDecoder) decode(buffer []byte, now time.Time) (Point, bool) {
	eventType := binary.LittleEndian.Uint16(buffer[16:18])
	code := binary.LittleEndian.Uint16(buffer[18:20])
	value := int32(binary.LittleEndian.Uint32(buffer[20:24]))
	switch eventType {
	case evAbs:
		if code == absMTPositionX {
			decoder.x = int(value)
		} else if code == absMTPositionY {
			decoder.y = int(value)
		}
	case evKey:
		if code == btnTouch {
			down := value == 1
			decoder.pendingRipple = decoder.pendingRipple || (down && !decoder.pressed)
			decoder.pressed = down
		}
	case evSyn:
		if code == synReport && decoder.pendingRipple {
			decoder.pendingRipple = false
			return Point{X: decoder.x, Y: decoder.y, At: now}, true
		}
	}
	return Point{}, false
}

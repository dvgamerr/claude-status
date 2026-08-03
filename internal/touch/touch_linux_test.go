//go:build linux

package touch

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func encodeEvent(eventType, code uint16, value int32) []byte {
	buffer := make([]byte, eventSize)
	binary.LittleEndian.PutUint16(buffer[16:18], eventType)
	binary.LittleEndian.PutUint16(buffer[18:20], code)
	binary.LittleEndian.PutUint32(buffer[20:24], uint32(value))
	return buffer
}

func TestWatchEmitsOnePointPerPressAndIgnoresRelease(t *testing.T) {
	var raw []byte
	raw = append(raw, encodeEvent(evAbs, absMTPositionX, 100)...)
	raw = append(raw, encodeEvent(evAbs, absMTPositionY, 200)...)
	raw = append(raw, encodeEvent(evKey, btnTouch, 1)...)
	raw = append(raw, encodeEvent(evSyn, synReport, 0)...)
	raw = append(raw, encodeEvent(evKey, btnTouch, 0)...)
	raw = append(raw, encodeEvent(evSyn, synReport, 0)...)

	path := filepath.Join(t.TempDir(), "fake-touch-device")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	points, err := Watch(ctx, path)
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	first, ok := <-points
	if !ok {
		t.Fatal("expected one point, channel closed with none")
	}
	if first.X != 100 || first.Y != 200 {
		t.Fatalf("Point = %+v, want X=100 Y=200", first)
	}

	select {
	case extra, ok := <-points:
		if ok {
			t.Fatalf("expected no second point (release should not emit), got %+v", extra)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("channel did not close after EOF within timeout")
	}
}

func TestWatchReturnsErrorForMissingDevice(t *testing.T) {
	if _, err := Watch(context.Background(), filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected an error opening a missing device")
	}
}

func TestWatchRejectsNilContextAndClosesOnCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty-touch-device")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	//lint:ignore SA1012 verifying Watch's own nil-context guard requires passing a literal nil
	if _, err := Watch(nil, path); err == nil {
		t.Fatal("expected nil context error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	points, err := Watch(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case _, ok := <-points:
		if ok {
			t.Fatal("unexpected point")
		}
	case <-time.After(time.Second):
		t.Fatal("touch channel did not close after cancellation")
	}
}

//go:build !linux

package touch

import (
	"context"
	"testing"
)

func TestWatchIsUnsupported(t *testing.T) {
	points, err := Watch(context.Background(), "/dev/input/event0")
	if points != nil || err == nil {
		t.Fatalf("Watch() = %v, %v", points, err)
	}
}

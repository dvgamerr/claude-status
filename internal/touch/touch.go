// Package touch reads raw touchscreen contacts so the dashboard can draw a
// tap ripple. It never touches provider data or the sanitized Snapshot —
// touch coordinates are a purely local UI concern and are never persisted
// or mirrored.
package touch

import "time"

// Point is one finger-down contact, in framebuffer pixel coordinates. The
// official Raspberry Pi touch display reports ABS_X/ABS_Y already mapped
// 1:1 to the panel's 800x480 pixels, so no scaling is needed here.
type Point struct {
	X, Y int
	At   time.Time
}

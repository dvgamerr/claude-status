// Package limitio provides bounded in-memory writers for subprocess diagnostics.
package limitio

import "bytes"

// DiagnosticLimit is the maximum stderr retained from one subprocess.
const DiagnosticLimit = 32 << 10

// Buffer retains at most limit bytes while reporting successful writes to the
// producer. This prevents a noisy subprocess from growing memory without bound.
type Buffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

// NewBuffer creates a bounded buffer. Non-positive limits retain no bytes.
func NewBuffer(limit int) *Buffer {
	if limit < 0 {
		limit = 0
	}
	return &Buffer{limit: limit}
}

func (buffer *Buffer) Write(data []byte) (int, error) {
	originalLength := len(data)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining < len(data) {
		buffer.truncated = true
		if remaining < 0 {
			remaining = 0
		}
		data = data[:remaining]
	}
	_, _ = buffer.buffer.Write(data)
	return originalLength, nil
}

func (buffer *Buffer) String() string {
	if !buffer.truncated {
		return buffer.buffer.String()
	}
	return buffer.buffer.String() + "\n[output truncated]"
}

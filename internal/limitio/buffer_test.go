package limitio

import "testing"

func TestBufferLimitsRetainedDataWithoutShortWrite(t *testing.T) {
	buffer := NewBuffer(4)
	n, err := buffer.Write([]byte("abcdef"))
	if err != nil || n != 6 {
		t.Fatalf("Write() = %d, %v", n, err)
	}
	if got := buffer.String(); got != "abcd\n[output truncated]" {
		t.Fatalf("String() = %q", got)
	}
	n, err = buffer.Write([]byte("gh"))
	if err != nil || n != 2 {
		t.Fatalf("second Write() = %d, %v", n, err)
	}
}

func TestBufferWithoutTruncation(t *testing.T) {
	buffer := NewBuffer(-1)
	if _, err := buffer.Write([]byte("discarded")); err != nil {
		t.Fatal(err)
	}
	if got := buffer.String(); got != "\n[output truncated]" {
		t.Fatalf("String() = %q", got)
	}

	buffer = NewBuffer(10)
	_, _ = buffer.Write([]byte("ok"))
	if got := buffer.String(); got != "ok" {
		t.Fatalf("String() = %q", got)
	}
}

// TestBufferWriteClampsNegativeRemaining exercises the defensive clamp for
// when the buffer already holds more than its limit's worth of bytes (not
// reachable through NewBuffer + Write alone, so the fixture is built by
// hand via direct field access within this package).
func TestBufferWriteClampsNegativeRemaining(t *testing.T) {
	buffer := &Buffer{limit: 2}
	buffer.buffer.WriteString("abcd")

	n, err := buffer.Write([]byte("ef"))
	if err != nil || n != 2 {
		t.Fatalf("Write() = %d, %v", n, err)
	}
	if got := buffer.String(); got != "abcd\n[output truncated]" {
		t.Fatalf("String() = %q", got)
	}
}

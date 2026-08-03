package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewWritesScopedPlainTextLog(t *testing.T) {
	var output bytes.Buffer
	logger := New(&output, "test-command")
	logger.Info().Str("key", "value").Msg("hello")
	got := output.String()
	for _, want := range []string{"INF", "hello", "cmd=test-command", "key=value"} {
		if !strings.Contains(got, want) {
			t.Fatalf("log output %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("log output contains ANSI color: %q", got)
	}
}

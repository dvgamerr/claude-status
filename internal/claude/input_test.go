package claude

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDecodeAndSanitizeSnapshot(t *testing.T) {
	raw := `{
      "session_id":"abc-123",
      "session_name":"demo\nterminal",
      "transcript_path":"/secret/conversation.jsonl",
      "prompt":"do not persist me",
      "model":{"id":"claude-opus","display_name":"Opus"},
      "version":"2.1.90",
      "context_window":{
        "total_input_tokens":15500,
        "total_output_tokens":1200,
        "context_window_size":200000,
        "used_percentage":108,
        "remaining_percentage":-4,
        "current_usage":{"input_tokens":8500,"cache_read_input_tokens":2000}
      },
      "rate_limits":{
        "five_hour":{"used_percentage":23.5,"resets_at":1738425600},
        "seven_day":null
      },
      "cost":{"total_cost_usd":0.42,"total_duration_ms":45000,"total_lines_added":12},
      "effort":{"level":"high"},
      "thinking":{"enabled":true}
    }`

	input, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.FixedZone("ICT", 7*60*60))
	snapshot := ToSnapshot(input, now)

	if snapshot.SchemaVersion != 1 || snapshot.Session.ID != "abc-123" {
		t.Fatalf("unexpected snapshot identity: %+v", snapshot)
	}
	if snapshot.Session.Name != "demoterminal" {
		t.Fatalf("control characters were not removed: %q", snapshot.Session.Name)
	}
	if got := *snapshot.Context.UsedPercentage; got != 100 {
		t.Fatalf("used percentage = %v, want 100", got)
	}
	if got := *snapshot.Context.RemainingPercentage; got != 0 {
		t.Fatalf("remaining percentage = %v, want 0", got)
	}
	if snapshot.RateLimits.SevenDay.UsedPercentage != nil {
		t.Fatal("null weekly rate limit should remain unavailable")
	}
	if snapshot.ThinkingEnabled == nil || !*snapshot.ThinkingEnabled {
		t.Fatal("thinking flag was not preserved")
	}
	if !snapshot.CapturedAt.Equal(now.UTC()) {
		t.Fatalf("captured_at = %s, want %s", snapshot.CapturedAt, now.UTC())
	}

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"transcript", "/secret", "prompt", "do not persist"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("sanitized snapshot contains %q: %s", forbidden, encoded)
		}
	}
}

func TestDecodeAllowsMissingAndNullFields(t *testing.T) {
	input, err := Decode(strings.NewReader(`{"session_id":null,"model":null,"context_window":null,"rate_limits":null}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	snapshot := ToSnapshot(input, time.Unix(1, 0))
	if snapshot.Session.ID != "unknown" {
		t.Fatalf("session ID = %q, want unknown", snapshot.Session.ID)
	}
	if got := StatusLine(snapshot); got != "[Claude] waiting for first response" {
		t.Fatalf("StatusLine() = %q", got)
	}
}

func TestToSnapshotSanitizesAndLimitsSessionID(t *testing.T) {
	input := Input{SessionID: "  abc\n\t" + strings.Repeat("x", maxSessionIDRunes+20) + "  "}
	snapshot := ToSnapshot(input, time.Unix(1, 0))
	if strings.ContainsAny(snapshot.Session.ID, "\n\t") {
		t.Fatalf("session ID still contains control characters: %q", snapshot.Session.ID)
	}
	if got := len([]rune(snapshot.Session.ID)); got != maxSessionIDRunes {
		t.Fatalf("session ID length = %d, want %d", got, maxSessionIDRunes)
	}
}

func TestStatusLine(t *testing.T) {
	input, err := Decode(strings.NewReader(`{
      "session_id":"x",
      "model":{"display_name":"Opus"},
      "context_window":{"used_percentage":38},
      "rate_limits":{"five_hour":{"used_percentage":23.5},"seven_day":{"used_percentage":41.2}}
    }`))
	if err != nil {
		t.Fatal(err)
	}
	got := StatusLine(ToSnapshot(input, time.Now()))
	want := "[Opus] 5h 24% · 7d 41% · ctx 38%"
	if got != want {
		t.Fatalf("StatusLine() = %q, want %q", got, want)
	}
}

func TestDecodeRejectsTrailingJSON(t *testing.T) {
	_, err := Decode(strings.NewReader(`{} {}`))
	if err == nil || !strings.Contains(err.Error(), "multiple JSON") {
		t.Fatalf("Decode() error = %v, want multiple JSON error", err)
	}
}

func TestDecodeRejectsOversizedInput(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"padding":"` + strings.Repeat("x", maxInputBytes) + `"}`))
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("Decode() error = %v, want size error", err)
	}
}

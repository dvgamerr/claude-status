package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecodeNotificationAllowlist(t *testing.T) {
	raw := `{
      "type":"agent-turn-complete",
      "thread-id":"019fb916-7c07-70b1-9462-2802d57220a2",
      "turn-id":"turn-1",
      "input-messages":["do not persist"],
      "last-assistant-message":"also private"
    }`
	notification, err := DecodeNotification(raw)
	if err != nil {
		t.Fatalf("DecodeNotification() error = %v", err)
	}
	if notification.Type != "agent-turn-complete" || notification.ThreadID != "019fb916-7c07-70b1-9462-2802d57220a2" {
		t.Fatalf("notification = %+v", notification)
	}
	encoded, err := json.Marshal(notification)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"do not persist", "also private", "input-messages"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("allowlisted notification contains %q: %s", forbidden, encoded)
		}
	}
}

func TestDecodeNotificationRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: " ", want: "empty"},
		{name: "invalid JSON", raw: "{", want: "decode"},
		{name: "multiple", raw: `{"thread-id":"x"} {}`, want: "multiple"},
		{name: "missing thread", raw: `{}`, want: "thread-id"},
		{name: "unsafe thread", raw: `{"thread-id":"../../secret"}`, want: "thread-id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeNotification(tt.raw)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodeNotification() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestSnapshotFromNotificationReadsOnlyUsageMetadata(t *testing.T) {
	home := t.TempDir()
	threadID := "019fb916-7c07-70b1-9462-2802d57220a2"
	rollout := filepath.Join(home, "sessions", "2026", "08", "01", "rollout-"+threadID+".jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	tooLongPrivateLine := `{"type":"response_item","payload":{"message":"` + strings.Repeat("private", maxRolloutLineBytes) + `"}}` + "\n"
	content := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"` + threadID + `","cli_version":"0.146.0","cwd":"C:/private/project"}}`,
		`{"type":"response_item","payload":{"type":"message","text":"private prompt"}}`,
		`{"type":"turn_context","payload":{"model":"gpt-5.6-sol","effort":"xhigh","cwd":"C:/private/project"}}`,
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":50000,"cached_input_tokens":32000,"output_tokens":2000,"reasoning_output_tokens":1000,"total_tokens":52000},"model_context_window":200000},"rate_limits":{"primary":{"used_percent":51,"window_minutes":300,"resets_at":1785500280},"secondary":{"used_percent":34,"window_minutes":10080,"resets_at":1785675600}}}}`,
	}, "\n") + "\n" + tooLongPrivateLine +
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":60000,"cached_input_tokens":40000,"output_tokens":4000,"total_tokens":64000},"model_context_window":200000},"rate_limits":{"primary":{"used_percent":52,"window_minutes":300,"resets_at":1785500281},"secondary":{"used_percent":35,"window_minutes":10080,"resets_at":1785675601}}}}` + "\n"
	if err := os.WriteFile(rollout, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.FixedZone("ICT", 7*60*60))
	snapshot, err := SnapshotFromNotification(Notification{ThreadID: threadID}, home, now)
	if err != nil {
		t.Fatalf("SnapshotFromNotification() error = %v", err)
	}
	if snapshot.Provider != "codex" || snapshot.Model.ID != "gpt-5.6-sol" || snapshot.Effort != "xhigh" {
		t.Fatalf("identity metadata = %+v", snapshot)
	}
	if snapshot.Context.UsedPercentage == nil || *snapshot.Context.UsedPercentage != 32 {
		t.Fatalf("context used = %v, want 32", snapshot.Context.UsedPercentage)
	}
	if snapshot.Context.TotalInputTokens == nil || *snapshot.Context.TotalInputTokens != 60000 || snapshot.Context.TotalOutputTokens == nil || *snapshot.Context.TotalOutputTokens != 4000 {
		t.Fatalf("token usage = %+v", snapshot.Context)
	}
	if snapshot.RateLimits.FiveHour.UsedPercentage == nil || *snapshot.RateLimits.FiveHour.UsedPercentage != 52 || snapshot.RateLimits.SevenDay.UsedPercentage == nil || *snapshot.RateLimits.SevenDay.UsedPercentage != 35 {
		t.Fatalf("rate limits = %+v", snapshot.RateLimits)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private prompt", "private project", "C:/private"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("snapshot contains private rollout data %q: %s", forbidden, encoded)
		}
	}
}

func TestSnapshotFromNotificationHandlesMissingAndReorderedWindows(t *testing.T) {
	home := t.TempDir()
	threadID := "thread_123"
	rollout := filepath.Join(home, "sessions", "rollout-"+threadID+".jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	content := `{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"output_tokens":5},"model_context_window":0},"rate_limits":{"primary":{"used_percent":150,"window_minutes":10080},"secondary":{"used_percent":-5,"window_minutes":300}}}}`
	if err := os.WriteFile(rollout, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := SnapshotFromNotification(Notification{ThreadID: threadID}, home, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Context.UsedPercentage != nil {
		t.Fatalf("zero context window produced percentage %v", snapshot.Context.UsedPercentage)
	}
	if got := *snapshot.RateLimits.SevenDay.UsedPercentage; got != 100 {
		t.Fatalf("7-day percentage = %v", got)
	}
	if got := *snapshot.RateLimits.FiveHour.UsedPercentage; got != 0 {
		t.Fatalf("5-hour percentage = %v", got)
	}
}

func TestSnapshotFromNotificationReportsMissingRollout(t *testing.T) {
	_, err := SnapshotFromNotification(Notification{ThreadID: "thread-404"}, t.TempDir(), time.Now())
	if err == nil || !strings.Contains(err.Error(), "no Codex rollout") {
		t.Fatalf("error = %v", err)
	}
}

func TestDefaultHomeHonorsEnvironment(t *testing.T) {
	want := filepath.Join(t.TempDir(), "codex-home")
	t.Setenv("CODEX_HOME", want)
	got, err := DefaultHome()
	if err != nil || got != filepath.Clean(want) {
		t.Fatalf("DefaultHome() = %q, %v; want %q", got, err, want)
	}
}

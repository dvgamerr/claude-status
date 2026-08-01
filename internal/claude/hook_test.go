package claude

import (
	"strings"
	"testing"

	"github.com/dvgamerr/claude-status/internal/model"
)

func TestDecodeHookAcceptsUnknownFieldsAndTrailingData(t *testing.T) {
	raw := `{"session_id":"abc","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"rm -rf /"},"transcript_path":"/secret.jsonl"}`
	hook, err := DecodeHook(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeHook() error = %v", err)
	}
	if hook.SessionID != "abc" || hook.HookEventName != "PreToolUse" {
		t.Fatalf("DecodeHook() = %+v", hook)
	}

	if _, err := DecodeHook(strings.NewReader(`{"session_id":"a"}{"session_id":"b"}`)); err == nil {
		t.Fatal("expected error for multiple JSON values")
	}
	if _, err := DecodeHook(strings.NewReader(`   `)); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestActivityForHook(t *testing.T) {
	cases := []struct {
		event   string
		message string
		want    string
		wantOK  bool
	}{
		{"UserPromptSubmit", "", model.ActivityWorking, true},
		{"PreToolUse", "", model.ActivityWorking, true},
		{"Stop", "", model.ActivityIdle, true},
		{"SubagentStop", "", model.ActivityIdle, true},
		{"Notification", "Claude needs your permission to use Bash", model.ActivityWaitingApproval, true},
		{"Notification", "Claude is waiting for your input", "", false},
		{"SessionStart", "", "", false},
		{"", "", "", false},
	}
	for _, testCase := range cases {
		state, ok := ActivityForHook(HookInput{HookEventName: testCase.event, Message: testCase.message})
		if state != testCase.want || ok != testCase.wantOK {
			t.Errorf("ActivityForHook(%q, %q) = (%q, %v), want (%q, %v)", testCase.event, testCase.message, state, ok, testCase.want, testCase.wantOK)
		}
	}
}

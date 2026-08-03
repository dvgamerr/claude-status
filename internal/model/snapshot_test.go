package model

import (
	"testing"
	"time"
)

func TestContextTokenFallbacks(t *testing.T) {
	input, cacheCreate, cacheRead, output := int64(10), int64(20), int64(30), int64(40)
	context := Context{CurrentUsage: TokenUsage{
		InputTokens: &input, CacheCreationInputTokens: &cacheCreate,
		CacheReadInputTokens: &cacheRead, OutputTokens: &output,
	}}
	if got := context.InputTokens(); got == nil || *got != 60 {
		t.Fatalf("InputTokens() = %v, want 60", got)
	}
	if got := context.OutputTokens(); got == nil || *got != 40 {
		t.Fatalf("OutputTokens() = %v, want 40", got)
	}
	totalInput, totalOutput := int64(100), int64(200)
	context.TotalInputTokens, context.TotalOutputTokens = &totalInput, &totalOutput
	if context.InputTokens() != &totalInput || context.OutputTokens() != &totalOutput {
		t.Fatal("explicit totals did not take precedence")
	}
	if (Context{}).InputTokens() != nil || (Context{}).OutputTokens() != nil {
		t.Fatal("empty context returned token counts")
	}
}

func TestCanonicalProvider(t *testing.T) {
	for input, want := range map[string]string{
		" CLAUDE ": ProviderClaude,
		"Codex":    ProviderCodex,
		"openai":   ProviderCodex,
		"unknown":  "",
	} {
		if got := CanonicalProvider(input); got != want {
			t.Errorf("CanonicalProvider(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSnapshotIsNewer(t *testing.T) {
	base := time.Unix(100, 0)
	current := Snapshot{CapturedAt: base, Activity: Activity{UpdatedAt: base}, Session: Session{ID: "a"}}
	tests := []struct {
		name      string
		candidate Snapshot
		want      bool
	}{
		{name: "capture", candidate: Snapshot{CapturedAt: base.Add(time.Second)}, want: true},
		{name: "activity", candidate: Snapshot{CapturedAt: base, Activity: Activity{UpdatedAt: base.Add(time.Second)}}, want: true},
		{name: "session tie", candidate: Snapshot{CapturedAt: base, Activity: Activity{UpdatedAt: base}, Session: Session{ID: "b"}}, want: true},
		{name: "older", candidate: Snapshot{CapturedAt: base.Add(-time.Second)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SnapshotIsNewer(tt.candidate, current); got != tt.want {
				t.Fatalf("SnapshotIsNewer() = %v, want %v", got, tt.want)
			}
		})
	}
}

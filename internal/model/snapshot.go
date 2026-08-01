package model

import "time"

const CurrentSchemaVersion = 1

// Activity states describe what the source session is doing right now, as
// reported by Claude Code hooks. They are derived, boolean-ish signals only
// — never the hook's raw message/prompt text.
const (
	ActivityWorking         = "working"
	ActivityIdle            = "idle"
	ActivityWaitingApproval = "waiting_approval"
)

// Snapshot is the deliberately small, sanitized representation persisted by
// claude-status. It must never contain prompts, transcripts, credentials, or
// arbitrary fields copied from Claude or Codex provider input.
type Snapshot struct {
	SchemaVersion int       `json:"schema_version"`
	CapturedAt    time.Time `json:"captured_at"`
	Provider      string    `json:"provider,omitempty"`
	ClientVersion string    `json:"client_version,omitempty"`
	Session       Session   `json:"session"`
	Model         Model     `json:"model"`
	// ClaudeCodeVersion is retained and populated for compatibility with older
	// snapshots/readers; provider-neutral consumers should use ClientVersion.
	ClaudeCodeVersion string     `json:"claude_code_version,omitempty"`
	Context           Context    `json:"context"`
	RateLimits        RateLimits `json:"rate_limits"`
	Cost              Cost       `json:"cost"`
	Effort            string     `json:"effort,omitempty"`
	ThinkingEnabled   *bool      `json:"thinking_enabled,omitempty"`
	Activity          Activity   `json:"activity,omitempty"`
}

// Activity is set by Claude Code hook events, independent of the statusLine
// refresh cycle, so the dashboard can animate work-in-progress vs idle vs a
// pending permission prompt.
type Activity struct {
	State     string    `json:"state,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Session struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type Model struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type Context struct {
	UsedPercentage      *float64   `json:"used_percentage,omitempty"`
	RemainingPercentage *float64   `json:"remaining_percentage,omitempty"`
	WindowSize          *int64     `json:"window_size,omitempty"`
	TotalInputTokens    *int64     `json:"total_input_tokens,omitempty"`
	TotalOutputTokens   *int64     `json:"total_output_tokens,omitempty"`
	CurrentUsage        TokenUsage `json:"current_usage"`
	Exceeds200KTokens   *bool      `json:"exceeds_200k_tokens,omitempty"`
}

type TokenUsage struct {
	InputTokens              *int64 `json:"input_tokens,omitempty"`
	OutputTokens             *int64 `json:"output_tokens,omitempty"`
	CacheCreationInputTokens *int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     *int64 `json:"cache_read_input_tokens,omitempty"`
}

type RateLimits struct {
	FiveHour  RateWindow `json:"five_hour"`
	SevenDay  RateWindow `json:"seven_day"`
	Plan      string     `json:"plan,omitempty"`
	Unlimited *bool      `json:"unlimited,omitempty"`
}

type RateWindow struct {
	UsedPercentage *float64 `json:"used_percentage,omitempty"`
	ResetsAt       *int64   `json:"resets_at,omitempty"`
}

type Cost struct {
	TotalCostUSD       *float64 `json:"total_cost_usd,omitempty"`
	TotalDurationMS    *int64   `json:"total_duration_ms,omitempty"`
	TotalAPIDurationMS *int64   `json:"total_api_duration_ms,omitempty"`
	TotalLinesAdded    *int64   `json:"total_lines_added,omitempty"`
	TotalLinesRemoved  *int64   `json:"total_lines_removed,omitempty"`
}

package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
)

const (
	maxInputBytes     = 4 << 20
	maxSessionIDRunes = 256
)

// Input models only the official statusLine fields used by this application.
// Unknown fields are intentionally accepted for forward compatibility and are
// discarded when a Snapshot is created.
type Input struct {
	SessionID     string         `json:"session_id"`
	SessionName   string         `json:"session_name"`
	Model         Model          `json:"model"`
	Version       string         `json:"version"`
	Cost          *Cost          `json:"cost"`
	ContextWindow *ContextWindow `json:"context_window"`
	RateLimits    *RateLimits    `json:"rate_limits"`
	Exceeds200K   *bool          `json:"exceeds_200k_tokens"`
	Effort        *Effort        `json:"effort"`
	Thinking      *Thinking      `json:"thinking"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Cost struct {
	TotalCostUSD       *float64 `json:"total_cost_usd"`
	TotalDurationMS    *int64   `json:"total_duration_ms"`
	TotalAPIDurationMS *int64   `json:"total_api_duration_ms"`
	TotalLinesAdded    *int64   `json:"total_lines_added"`
	TotalLinesRemoved  *int64   `json:"total_lines_removed"`
}

type ContextWindow struct {
	TotalInputTokens    *int64      `json:"total_input_tokens"`
	TotalOutputTokens   *int64      `json:"total_output_tokens"`
	ContextWindowSize   *int64      `json:"context_window_size"`
	UsedPercentage      *float64    `json:"used_percentage"`
	RemainingPercentage *float64    `json:"remaining_percentage"`
	CurrentUsage        *TokenUsage `json:"current_usage"`
}

type TokenUsage struct {
	InputTokens              *int64 `json:"input_tokens"`
	OutputTokens             *int64 `json:"output_tokens"`
	CacheCreationInputTokens *int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int64 `json:"cache_read_input_tokens"`
}

type RateLimits struct {
	FiveHour *RateWindow `json:"five_hour"`
	SevenDay *RateWindow `json:"seven_day"`
}

type RateWindow struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       *int64   `json:"resets_at"`
}

type Effort struct {
	Level string `json:"level"`
}

type Thinking struct {
	Enabled *bool `json:"enabled"`
}

func Decode(r io.Reader) (Input, error) {
	var input Input
	data, err := io.ReadAll(io.LimitReader(r, maxInputBytes+1))
	if err != nil {
		return input, fmt.Errorf("read statusLine input: %w", err)
	}
	if len(data) > maxInputBytes {
		return input, fmt.Errorf("statusLine input exceeds %d bytes", maxInputBytes)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return input, errors.New("statusLine input is empty")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&input); err != nil {
		return input, fmt.Errorf("decode statusLine JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return input, errors.New("statusLine input contains multiple JSON values")
		}
		return input, fmt.Errorf("decode trailing statusLine data: %w", err)
	}
	return input, nil
}

func ToSnapshot(input Input, now time.Time) model.Snapshot {
	sessionID := cleanText(input.SessionID, maxSessionIDRunes)
	if sessionID == "" {
		sessionID = "unknown"
	}

	snapshot := model.Snapshot{
		SchemaVersion:     model.CurrentSchemaVersion,
		CapturedAt:        now.UTC(),
		Session:           model.Session{ID: sessionID, Name: cleanText(input.SessionName, 120)},
		Model:             model.Model{ID: cleanText(input.Model.ID, 120), DisplayName: cleanText(input.Model.DisplayName, 120)},
		ClaudeCodeVersion: cleanText(input.Version, 80),
		Context:           model.Context{Exceeds200KTokens: copyBool(input.Exceeds200K)},
	}

	if input.ContextWindow != nil {
		context := input.ContextWindow
		snapshot.Context.UsedPercentage = percentage(context.UsedPercentage)
		snapshot.Context.RemainingPercentage = percentage(context.RemainingPercentage)
		snapshot.Context.WindowSize = nonNegativeInt(context.ContextWindowSize)
		snapshot.Context.TotalInputTokens = nonNegativeInt(context.TotalInputTokens)
		snapshot.Context.TotalOutputTokens = nonNegativeInt(context.TotalOutputTokens)
		if context.CurrentUsage != nil {
			snapshot.Context.CurrentUsage = model.TokenUsage{
				InputTokens:              nonNegativeInt(context.CurrentUsage.InputTokens),
				OutputTokens:             nonNegativeInt(context.CurrentUsage.OutputTokens),
				CacheCreationInputTokens: nonNegativeInt(context.CurrentUsage.CacheCreationInputTokens),
				CacheReadInputTokens:     nonNegativeInt(context.CurrentUsage.CacheReadInputTokens),
			}
		}
	}

	if input.RateLimits != nil {
		snapshot.RateLimits.FiveHour = convertRateWindow(input.RateLimits.FiveHour)
		snapshot.RateLimits.SevenDay = convertRateWindow(input.RateLimits.SevenDay)
	}

	if input.Cost != nil {
		snapshot.Cost = model.Cost{
			TotalCostUSD:       nonNegativeFloat(input.Cost.TotalCostUSD),
			TotalDurationMS:    nonNegativeInt(input.Cost.TotalDurationMS),
			TotalAPIDurationMS: nonNegativeInt(input.Cost.TotalAPIDurationMS),
			TotalLinesAdded:    nonNegativeInt(input.Cost.TotalLinesAdded),
			TotalLinesRemoved:  nonNegativeInt(input.Cost.TotalLinesRemoved),
		}
	}
	if input.Effort != nil {
		snapshot.Effort = cleanText(input.Effort.Level, 24)
	}
	if input.Thinking != nil {
		snapshot.ThinkingEnabled = copyBool(input.Thinking.Enabled)
	}

	return snapshot
}

func StatusLine(snapshot model.Snapshot) string {
	name := snapshot.Model.DisplayName
	if name == "" {
		name = snapshot.Model.ID
	}
	if name == "" {
		name = "Claude"
	}

	parts := make([]string, 0, 3)
	if p := snapshot.RateLimits.FiveHour.UsedPercentage; p != nil {
		parts = append(parts, fmt.Sprintf("5h %.0f%%", *p))
	}
	if p := snapshot.RateLimits.SevenDay.UsedPercentage; p != nil {
		parts = append(parts, fmt.Sprintf("7d %.0f%%", *p))
	}
	if p := snapshot.Context.UsedPercentage; p != nil {
		parts = append(parts, fmt.Sprintf("ctx %.0f%%", *p))
	}
	if len(parts) == 0 {
		parts = append(parts, "waiting for first response")
	}
	return fmt.Sprintf("[%s] %s", name, strings.Join(parts, " · "))
}

func convertRateWindow(input *RateWindow) model.RateWindow {
	if input == nil {
		return model.RateWindow{}
	}
	return model.RateWindow{
		UsedPercentage: percentage(input.UsedPercentage),
		ResetsAt:       positiveInt(input.ResetsAt),
	}
}

func percentage(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	v := min(100, max(0, *value))
	return &v
}

func nonNegativeFloat(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
		return nil
	}
	v := *value
	return &v
}

func nonNegativeInt(value *int64) *int64 {
	if value == nil || *value < 0 {
		return nil
	}
	v := *value
	return &v
}

func positiveInt(value *int64) *int64 {
	if value == nil || *value <= 0 {
		return nil
	}
	v := *value
	return &v
}

func copyBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func cleanText(value string, limit int) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[:limit])
	}
	return value
}

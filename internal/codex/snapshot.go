package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dvgamerr/claude-status/internal/model"
)

const (
	maxNotificationBytes = 4 << 20
	maxRolloutLineBytes  = 256 << 10
)

var threadIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type Notification struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread-id"`
	TurnID   string `json:"turn-id"`
}

type tokenUsage struct {
	InputTokens       *int64 `json:"input_tokens"`
	CachedInputTokens *int64 `json:"cached_input_tokens"`
	OutputTokens      *int64 `json:"output_tokens"`
	TotalTokens       *int64 `json:"total_tokens"`
}

type rateWindow struct {
	UsedPercent   *float64 `json:"used_percent"`
	WindowMinutes *int64   `json:"window_minutes"`
	ResetsAt      *int64   `json:"resets_at"`
}

type rolloutValues struct {
	SessionID     string
	ClientVersion string
	Model         string
	Effort        string
	Usage         tokenUsage
	ContextWindow *int64
	Primary       *rateWindow
	Secondary     *rateWindow
	Plan          string
	Unlimited     *bool
}

func DecodeNotification(value string) (Notification, error) {
	var notification Notification
	if len(value) > maxNotificationBytes {
		return notification, fmt.Errorf("Codex notification exceeds %d bytes", maxNotificationBytes)
	}
	if strings.TrimSpace(value) == "" {
		return notification, errors.New("Codex notification is empty")
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	if err := decoder.Decode(&notification); err != nil {
		return notification, fmt.Errorf("decode Codex notification JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return notification, errors.New("Codex notification contains multiple JSON values")
		}
		return notification, fmt.Errorf("decode trailing Codex notification data: %w", err)
	}
	notification.Type = cleanText(notification.Type, 80)
	notification.ThreadID = cleanText(notification.ThreadID, 128)
	notification.TurnID = cleanText(notification.TurnID, 128)
	if !threadIDPattern.MatchString(notification.ThreadID) {
		return notification, errors.New("Codex notification has no valid thread-id")
	}
	return notification, nil
}

// SnapshotFromNotification reads the matching local Codex rollout but copies
// only usage/model metadata into the returned Snapshot. Prompts, responses,
// paths, tool calls, and arbitrary rollout fields are never retained.
func SnapshotFromNotification(notification Notification, codexHome string, now time.Time) (model.Snapshot, error) {
	if !threadIDPattern.MatchString(notification.ThreadID) {
		return model.Snapshot{}, errors.New("Codex notification has no valid thread-id")
	}
	rolloutPath, err := findRollout(codexHome, notification.ThreadID)
	if err != nil {
		return model.Snapshot{}, err
	}
	values, err := readRollout(rolloutPath)
	if err != nil {
		return model.Snapshot{}, err
	}
	if values.SessionID == "" {
		values.SessionID = notification.ThreadID
	}

	snapshot := model.Snapshot{
		SchemaVersion: model.CurrentSchemaVersion,
		CapturedAt:    now.UTC(),
		Provider:      "codex",
		ClientVersion: cleanText(values.ClientVersion, 80),
		Session:       model.Session{ID: cleanText(values.SessionID, 128)},
		Model: model.Model{
			ID:          cleanText(values.Model, 120),
			DisplayName: cleanText(values.Model, 120),
		},
		Effort: cleanText(values.Effort, 24),
	}

	snapshot.Context.WindowSize = nonNegativeInt(values.ContextWindow)
	snapshot.Context.TotalInputTokens = nonNegativeInt(values.Usage.InputTokens)
	snapshot.Context.TotalOutputTokens = nonNegativeInt(values.Usage.OutputTokens)
	snapshot.Context.CurrentUsage.InputTokens = nonNegativeInt(values.Usage.InputTokens)
	snapshot.Context.CurrentUsage.OutputTokens = nonNegativeInt(values.Usage.OutputTokens)
	snapshot.Context.CurrentUsage.CacheReadInputTokens = nonNegativeInt(values.Usage.CachedInputTokens)
	usedTokens := values.Usage.TotalTokens
	if usedTokens == nil {
		usedTokens = sum(values.Usage.InputTokens, values.Usage.OutputTokens)
	}
	if usedTokens != nil && values.ContextWindow != nil && *values.ContextWindow > 0 {
		used := clampPercentage(float64(*usedTokens) / float64(*values.ContextWindow) * 100)
		remaining := 100 - used
		snapshot.Context.UsedPercentage = &used
		snapshot.Context.RemainingPercentage = &remaining
	}

	assignRateWindow(&snapshot.RateLimits, values.Primary, true)
	assignRateWindow(&snapshot.RateLimits, values.Secondary, false)
	snapshot.RateLimits.Plan = cleanText(values.Plan, 40)
	if values.Unlimited != nil {
		unlimited := *values.Unlimited
		snapshot.RateLimits.Unlimited = &unlimited
	}
	return snapshot, nil
}

func DefaultHome() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return filepath.Clean(value), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find Codex home: %w", err)
	}
	return filepath.Join(home, ".codex"), nil
}

func findRollout(codexHome, threadID string) (string, error) {
	root := filepath.Join(filepath.Clean(codexHome), "sessions")
	var selected string
	var selectedMod time.Time
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" || !strings.Contains(entry.Name(), threadID) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if selected == "" || info.ModTime().After(selectedMod) {
			selected = path
			selectedMod = info.ModTime()
		}
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("no Codex rollout found for thread %s", threadID)
	}
	if err != nil {
		return "", fmt.Errorf("find Codex rollout for thread %s: %w", threadID, err)
	}
	if selected == "" {
		return "", fmt.Errorf("no Codex rollout found for thread %s", threadID)
	}
	return selected, nil
}

func readRollout(path string) (rolloutValues, error) {
	var values rolloutValues
	file, err := os.Open(path)
	if err != nil {
		return values, fmt.Errorf("open Codex rollout: %w", err)
	}
	defer file.Close()

	err = forEachLimitedLine(file, maxRolloutLineBytes, func(line []byte) {
		var probe struct {
			Type    string `json:"type"`
			Payload struct {
				Type string `json:"type"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &probe) != nil {
			return
		}
		switch probe.Type {
		case "session_meta":
			var record struct {
				Payload struct {
					ID         string `json:"id"`
					CLIVersion string `json:"cli_version"`
				} `json:"payload"`
			}
			if json.Unmarshal(line, &record) == nil {
				values.SessionID = cleanText(record.Payload.ID, 128)
				values.ClientVersion = cleanText(record.Payload.CLIVersion, 80)
			}
		case "turn_context":
			var record struct {
				Payload struct {
					Model  string `json:"model"`
					Effort string `json:"effort"`
				} `json:"payload"`
			}
			if json.Unmarshal(line, &record) == nil {
				values.Model = cleanText(record.Payload.Model, 120)
				values.Effort = cleanText(record.Payload.Effort, 24)
			}
		case "event_msg":
			if probe.Payload.Type != "token_count" {
				return
			}
			var record struct {
				Payload struct {
					Info struct {
						LastTokenUsage     tokenUsage `json:"last_token_usage"`
						ModelContextWindow *int64     `json:"model_context_window"`
					} `json:"info"`
					RateLimits struct {
						Primary   *rateWindow `json:"primary"`
						Secondary *rateWindow `json:"secondary"`
						PlanType  string      `json:"plan_type"`
						Credits   *struct {
							Unlimited *bool `json:"unlimited"`
						} `json:"credits"`
					} `json:"rate_limits"`
				} `json:"payload"`
			}
			if json.Unmarshal(line, &record) == nil {
				values.Usage = record.Payload.Info.LastTokenUsage
				values.ContextWindow = record.Payload.Info.ModelContextWindow
				values.Primary = record.Payload.RateLimits.Primary
				values.Secondary = record.Payload.RateLimits.Secondary
				values.Plan = record.Payload.RateLimits.PlanType
				if record.Payload.RateLimits.Credits != nil {
					values.Unlimited = record.Payload.RateLimits.Credits.Unlimited
				}
			}
		}
	})
	if err != nil {
		return values, fmt.Errorf("read Codex rollout: %w", err)
	}
	return values, nil
}

func forEachLimitedLine(input io.Reader, limit int, visit func([]byte)) error {
	reader := bufio.NewReaderSize(input, 64<<10)
	buffer := make([]byte, 0, min(limit, 64<<10))
	overLimit := false
	for {
		fragment, err := reader.ReadSlice('\n')
		if !overLimit {
			if len(buffer)+len(fragment) <= limit {
				buffer = append(buffer, fragment...)
			} else {
				overLimit = true
				buffer = buffer[:0]
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if !overLimit {
			line := bytes.TrimSpace(buffer)
			if len(line) > 0 {
				visit(line)
			}
		}
		buffer = buffer[:0]
		overLimit = false
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func assignRateWindow(limits *model.RateLimits, input *rateWindow, primary bool) {
	if input == nil {
		return
	}
	window := model.RateWindow{
		UsedPercentage: percentage(input.UsedPercent),
		ResetsAt:       positiveInt(input.ResetsAt),
	}
	minutes := int64(0)
	if input.WindowMinutes != nil {
		minutes = *input.WindowMinutes
	}
	switch {
	case minutes >= 240 && minutes <= 360:
		limits.FiveHour = window
	case minutes >= 7*24*60-60 && minutes <= 7*24*60+60:
		limits.SevenDay = window
	case primary:
		limits.FiveHour = window
	default:
		limits.SevenDay = window
	}
}

func sum(values ...*int64) *int64 {
	found := false
	total := int64(0)
	for _, value := range values {
		if value != nil && *value >= 0 {
			found = true
			total += *value
		}
	}
	if !found {
		return nil
	}
	return &total
}

func percentage(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	result := clampPercentage(*value)
	return &result
}

func clampPercentage(value float64) float64 {
	return min(100, max(0, value))
}

func nonNegativeInt(value *int64) *int64 {
	if value == nil || *value < 0 {
		return nil
	}
	result := *value
	return &result
}

func positiveInt(value *int64) *int64 {
	if value == nil || *value <= 0 {
		return nil
	}
	result := *value
	return &result
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
		return string(runes[:limit])
	}
	return value
}

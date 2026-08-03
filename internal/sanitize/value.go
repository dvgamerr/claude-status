// Package sanitize contains the shared allowlisting primitives used when
// provider-owned data is copied into the persisted snapshot schema.
package sanitize

import (
	"math"
	"strings"
)

// Text trims whitespace, removes ASCII control characters, and limits the
// result to at most limit Unicode code points.
func Text(value string, limit int) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}

// ClampPercentage converts non-finite values to zero and otherwise clamps a
// percentage to the inclusive range 0..100.
func ClampPercentage(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return min(100, max(0, value))
}

// Percentage returns a sanitized copy of value, or nil for nil/non-finite
// input.
func Percentage(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	result := ClampPercentage(*value)
	return &result
}

// NonNegativeFloat64 returns a copy of a finite, non-negative value.
func NonNegativeFloat64(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
		return nil
	}
	result := *value
	return &result
}

// NonNegativeInt64 returns a copy of a non-negative value.
func NonNegativeInt64(value *int64) *int64 {
	if value == nil || *value < 0 {
		return nil
	}
	result := *value
	return &result
}

// PositiveInt64 returns a copy of a strictly positive value.
func PositiveInt64(value *int64) *int64 {
	if value == nil || *value <= 0 {
		return nil
	}
	result := *value
	return &result
}

// Bool returns an independent copy of value.
func Bool(value *bool) *bool {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

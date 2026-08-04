package sanitize

import (
	"math"
	"testing"
)

func TestText(t *testing.T) {
	if got := Text("  a\x00b\nçd  ", 3); got != "abç" {
		t.Fatalf("Text() = %q, want %q", got, "abç")
	}
	if got := Text("value", 0); got != "" {
		t.Fatalf("Text() with zero limit = %q", got)
	}
	if got := Text("value", -1); got != "" {
		t.Fatalf("Text() with negative limit = %q", got)
	}
	if got := Text("  ab  ", 10); got != "ab" {
		t.Fatalf("Text() with limit exceeding length = %q, want %q", got, "ab")
	}
	if got := Text("abc", 3); got != "abc" {
		t.Fatalf("Text() with limit equal to length = %q, want %q", got, "abc")
	}
	if got := Text("", 5); got != "" {
		t.Fatalf("Text() with empty input = %q", got)
	}
}

func TestNumericAndBooleanCopies(t *testing.T) {
	negative := int64(-1)
	zero := int64(0)
	positive := int64(7)
	if NonNegativeInt64(nil) != nil || NonNegativeInt64(&negative) != nil || PositiveInt64(&zero) != nil {
		t.Fatal("integer sanitizers accepted invalid values")
	}
	if got := NonNegativeInt64(&zero); got == nil || *got != 0 || got == &zero {
		t.Fatalf("NonNegativeInt64() = %v", got)
	}
	if got := PositiveInt64(&positive); got == nil || *got != positive || got == &positive {
		t.Fatalf("PositiveInt64() = %v", got)
	}

	truth := true
	if Bool(nil) != nil {
		t.Fatal("Bool(nil) returned a value")
	}
	if got := Bool(&truth); got == nil || !*got || got == &truth {
		t.Fatalf("Bool() = %v", got)
	}
}

func TestFloatingPointSanitizers(t *testing.T) {
	negative := -1.0
	valid := 12.5
	nan := math.NaN()
	inf := math.Inf(1)
	if Percentage(nil) != nil || Percentage(&nan) != nil || Percentage(&inf) != nil {
		t.Fatal("Percentage accepted nil or non-finite values")
	}
	if got := Percentage(&negative); got == nil || *got != 0 {
		t.Fatalf("Percentage(-1) = %v", got)
	}
	if got := Percentage(&valid); got == nil || *got != valid || got == &valid {
		t.Fatalf("Percentage(valid) = %v", got)
	}
	if NonNegativeFloat64(&negative) != nil || NonNegativeFloat64(&nan) != nil || NonNegativeFloat64(&inf) != nil {
		t.Fatal("NonNegativeFloat64 accepted invalid values")
	}
	if got := NonNegativeFloat64(&valid); got == nil || *got != valid || got == &valid {
		t.Fatalf("NonNegativeFloat64(valid) = %v", got)
	}
	if ClampPercentage(-2) != 0 || ClampPercentage(200) != 100 || ClampPercentage(math.NaN()) != 0 {
		t.Fatal("ClampPercentage did not clamp edge cases")
	}
}

package svganim

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

// -- Evaluate / parseSVG error paths -----------------------------------

func TestEvaluateErrorsOnEmptyInput(t *testing.T) {
	_, err := Evaluate([]byte(""), time.Now())
	if err == nil {
		t.Fatal("Evaluate: want error on empty input, got nil")
	}
	if !strings.Contains(err.Error(), "no root element") {
		t.Errorf("Evaluate error = %q, want it to mention \"no root element\"", err.Error())
	}
}

func TestParseSVGErrorsOnEmptyInput(t *testing.T) {
	_, err := parseSVG([]byte(""))
	if err == nil {
		t.Fatal("parseSVG: want error on empty input, got nil")
	}
	if !strings.Contains(err.Error(), "no root element") {
		t.Errorf("parseSVG error = %q, want it to mention \"no root element\"", err.Error())
	}
}

func TestParseSVGErrorsOnMalformedXML(t *testing.T) {
	// Mismatched close tag: a genuine decode-time xml.SyntaxError, not
	// merely a missing root element.
	_, err := parseSVG([]byte(`<svg><g></svg>`))
	if err == nil {
		t.Fatal("parseSVG: want error on mismatched tags, got nil")
	}
}

// -- evaluateNode: <set> element handling -------------------------------

func TestEvaluateNodeAppliesSetElement(t *testing.T) {
	root, err := parseSVG([]byte(`<g id="x"><set attributeName="opacity" to="0.5"/></g>`))
	if err != nil {
		t.Fatalf("parseSVG: %v", err)
	}
	evaluateNode(root, time.Now())
	if got := getAttr(root, "opacity"); got != "0.5" {
		t.Errorf("opacity = %q, want %q", got, "0.5")
	}
	if len(root.children) != 0 {
		t.Errorf("the <set> element should have been stripped, children = %v", root.children)
	}
}

func TestEvaluateNodeSkipsSetWithoutAttributeName(t *testing.T) {
	root, err := parseSVG([]byte(`<g id="x"><set to="0.5"/></g>`))
	if err != nil {
		t.Fatalf("parseSVG: %v", err)
	}
	evaluateNode(root, time.Now())
	if got := getAttr(root, "to"); got != "" {
		t.Errorf("a <set> without attributeName must not apply anything, got attribute %q", got)
	}
	if len(root.children) != 0 {
		t.Errorf("the <set> element should still be stripped even without attributeName, children = %v", root.children)
	}
}

// -- applyAttr: overwrite of an already-present attribute ---------------

func TestApplyAttrOverwritesExistingNonTransformAttr(t *testing.T) {
	n := &node{attrs: []xml.Attr{{Name: xml.Name{Local: "opacity"}, Value: "1"}}}
	applyAttr(n, "opacity", "0.5")
	if got := getAttr(n, "opacity"); got != "0.5" {
		t.Errorf("opacity = %q, want %q", got, "0.5")
	}
	if len(n.attrs) != 1 {
		t.Errorf("applyAttr should overwrite in place, not append; attrs = %v", n.attrs)
	}
}

func TestApplyAttrOverwritesEmptyExistingTransform(t *testing.T) {
	n := &node{attrs: []xml.Attr{{Name: xml.Name{Local: "transform"}, Value: ""}}}
	applyAttr(n, "transform", "rotate(5)")
	if got := getAttr(n, "transform"); got != "rotate(5)" {
		t.Errorf("an empty pre-existing transform must be overwritten, not composed with a leading space: transform = %q", got)
	}
}

func TestApplyAttrAppendsWhenAbsent(t *testing.T) {
	n := &node{}
	applyAttr(n, "opacity", "0.5")
	if got := getAttr(n, "opacity"); got != "0.5" {
		t.Errorf("opacity = %q, want %q", got, "0.5")
	}
}

// -- evaluateAnimate: early-out branches ---------------------------------

func mustParseElement(t *testing.T, src string) *node {
	t.Helper()
	n, err := parseSVG([]byte(src))
	if err != nil {
		t.Fatalf("parseSVG(%q): %v", src, err)
	}
	return n
}

func TestEvaluateAnimateMissingAttributeName(t *testing.T) {
	el := mustParseElement(t, `<animate values="0;1" dur="1s"/>`)
	_, _, ok := evaluateAnimate(el, time.Now())
	if ok {
		t.Error("evaluateAnimate: want ok=false when attributeName is missing")
	}
}

func TestEvaluateAnimateInvalidDur(t *testing.T) {
	el := mustParseElement(t, `<animate attributeName="opacity" values="0;1" dur="not-a-duration"/>`)
	_, _, ok := evaluateAnimate(el, time.Now())
	if ok {
		t.Error("evaluateAnimate: want ok=false when dur fails to parse")
	}
}

func TestEvaluateAnimateZeroDur(t *testing.T) {
	el := mustParseElement(t, `<animate attributeName="opacity" values="0;1" dur="0s"/>`)
	_, _, ok := evaluateAnimate(el, time.Now())
	if ok {
		t.Error("evaluateAnimate: want ok=false when dur<=0")
	}
}

func TestEvaluateAnimateKeyTimesCountMismatch(t *testing.T) {
	el := mustParseElement(t, `<animate attributeName="opacity" values="0;0.5;1" keyTimes="0;1" dur="1s"/>`)
	_, _, ok := evaluateAnimate(el, time.Now())
	if ok {
		t.Error("evaluateAnimate: want ok=false when keyTimes count doesn't match the number of keyframes")
	}
}

// -- loopPhase: finite repeatCount edge cases ----------------------------

func TestLoopPhaseElapsedBeforeBeginClampsToZero(t *testing.T) {
	now := time.Unix(0, 0)
	got := loopPhase(now, time.Second, 2*time.Second, "2")
	if got != 0 {
		t.Errorf("loopPhase before begin with finite repeatCount = %v, want 0", got)
	}
}

func TestLoopPhaseGarbageRepeatCountFallsBackToIndefinite(t *testing.T) {
	now := time.Unix(0, 0).Add(1500 * time.Millisecond)
	got := loopPhase(now, time.Second, 0, "not-a-number")
	want := 0.5
	if got != want {
		t.Errorf("loopPhase with unparsable repeatCount = %v, want %v (indefinite fallback)", got, want)
	}
}

// -- formatTransform: rotate/scale short forms and unknown type ---------

func TestFormatTransformRotateSingleArg(t *testing.T) {
	got, ok := formatTransform("rotate", []float64{45})
	if !ok || got != "rotate(45)" {
		t.Errorf("formatTransform(rotate, [45]) = (%q, %v), want (\"rotate(45)\", true)", got, ok)
	}
}

func TestFormatTransformScaleSingleArg(t *testing.T) {
	got, ok := formatTransform("scale", []float64{2})
	if !ok || got != "scale(2)" {
		t.Errorf("formatTransform(scale, [2]) = (%q, %v), want (\"scale(2)\", true)", got, ok)
	}
}

func TestFormatTransformScaleTwoArgs(t *testing.T) {
	got, ok := formatTransform("scale", []float64{2, 3})
	if !ok || got != "scale(2,3)" {
		t.Errorf("formatTransform(scale, [2,3]) = (%q, %v), want (\"scale(2,3)\", true)", got, ok)
	}
}

func TestFormatTransformUnknownType(t *testing.T) {
	got, ok := formatTransform("skewX", []float64{10})
	if ok || got != "" {
		t.Errorf("formatTransform(skewX, [10]) = (%q, %v), want (\"\", false)", got, ok)
	}
}

// -- clamp01 boundaries ---------------------------------------------------

func TestClamp01Boundaries(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{-1, 0},
		{-0.0001, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{1.0001, 1},
		{5, 1},
	}
	for _, tt := range cases {
		if got := clamp01(tt.in); got != tt.want {
			t.Errorf("clamp01(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// -- parseClockValue: every branch, malformed and edge-case forms -------

func TestParseClockValueBranches(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"empty", "", 0, true},
		{"whitespace only", "   ", 0, true},
		{"milliseconds", "600ms", 600 * time.Millisecond, false},
		{"milliseconds garbage", "abcms", 0, true},
		{"seconds", "0.6s", 600 * time.Millisecond, false},
		{"seconds garbage", "abcs", 0, true},
		{"bare number seconds", "2", 2 * time.Second, false},
		{"bare negative number", "-2", -2 * time.Second, false},
		{"negative seconds suffix", "-500ms", -500 * time.Millisecond, false},
		{"garbage", "not-a-clock-value", 0, true},
		{"hour:min:sec form unsupported", "01:30:00", 0, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseClockValue(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseClockValue(%q) = %v, nil; want an error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseClockValue(%q): unexpected error %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("parseClockValue(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// -- parseKeyframes / parseKeyTimes / parseFloatList error branches -----

func TestParseKeyframesInvalidNumberInKeyframe(t *testing.T) {
	_, err := parseKeyframes("0,0;abc,1")
	if err == nil {
		t.Fatal("parseKeyframes: want error on an invalid number inside a keyframe")
	}
}

func TestParseKeyTimesInvalidNumber(t *testing.T) {
	_, err := parseKeyTimes("a;b", 2)
	if err == nil {
		t.Fatal("parseKeyTimes: want error on invalid numbers")
	}
}

func TestParseKeyTimesCountMismatch(t *testing.T) {
	_, err := parseKeyTimes("0;1", 3)
	if err == nil {
		t.Fatal("parseKeyTimes: want error when entry count doesn't match n")
	}
	if !strings.Contains(err.Error(), "want 3") {
		t.Errorf("parseKeyTimes error = %q, want it to mention the expected count", err.Error())
	}
}

func TestParseFloatListEmpty(t *testing.T) {
	_, err := parseFloatList("")
	if err == nil {
		t.Fatal("parseFloatList(\"\"): want error, got nil")
	}
}

func TestParseFloatListOnlySeparators(t *testing.T) {
	_, err := parseFloatList("  ,  ,\t")
	if err == nil {
		t.Fatal("parseFloatList of only separators: want error, got nil")
	}
}

func TestParseFloatListInvalidNumber(t *testing.T) {
	_, err := parseFloatList("1,abc,3")
	if err == nil {
		t.Fatal("parseFloatList: want error on an invalid number")
	}
}

func TestParseFloatListValid(t *testing.T) {
	got, err := parseFloatList("1, 2,\t3")
	if err != nil {
		t.Fatalf("parseFloatList: unexpected error %v", err)
	}
	want := []float64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("parseFloatList = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parseFloatList[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// -- writeNode: text content, escaping, and attrName namespaces ---------

func TestWriteNodeEscapesTextContent(t *testing.T) {
	n := &node{name: "text", text: `A & B < C > "D"`}
	var buf bytes.Buffer
	writeNode(&buf, n)
	got := buf.String()
	want := `<text>A &amp; B &lt; C &gt; "D"</text>`
	if got != want {
		t.Errorf("writeNode text output = %q, want %q", got, want)
	}
}

func TestWriteNodeSkipsWhitespaceOnlyText(t *testing.T) {
	n := &node{name: "g", text: "\n   \t  "}
	var buf bytes.Buffer
	writeNode(&buf, n)
	got := buf.String()
	want := `<g/>`
	if got != want {
		t.Errorf("writeNode with whitespace-only text = %q, want self-closing %q", got, want)
	}
}

func TestAttrNameNamespaces(t *testing.T) {
	cases := []struct {
		name xml.Name
		want string
	}{
		{xml.Name{Local: "id"}, "id"},
		{xml.Name{Space: "xmlns", Local: "foo"}, "xmlns:foo"},
		{xml.Name{Space: "xlink", Local: "href"}, "xlink:href"},
		{xml.Name{Space: "http://www.w3.org/1999/xlink", Local: "href"}, "http://www.w3.org/1999/xlink:href"},
	}
	for _, tt := range cases {
		if got := attrName(tt.name); got != tt.want {
			t.Errorf("attrName(%+v) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// -- escapeText: direct coverage of every replaced character -------------

func TestEscapeText(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"plain", "plain"},
		{"a & b", "a &amp; b"},
		{"<tag>", "&lt;tag&gt;"},
		{"a & b < c > d", "a &amp; b &lt; c &gt; d"},
		// escapeText deliberately does not escape quotes; that's escapeAttr's job.
		{`"quoted"`, `"quoted"`},
	}
	for _, tt := range cases {
		if got := escapeText(tt.in); got != tt.want {
			t.Errorf("escapeText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

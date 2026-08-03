package svganim

import (
	"strings"
	"testing"
	"time"
)

func findByID(n *node, id string) *node {
	if getAttr(n, "id") == id {
		return n
	}
	for _, c := range n.children {
		if found := findByID(c, id); found != nil {
			return found
		}
	}
	return nil
}

func evalAttr(t *testing.T, source []byte, at time.Time, id, attr string) string {
	t.Helper()
	out, err := Evaluate(source, at)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	root, err := parseSVG(out)
	if err != nil {
		t.Fatalf("parseSVG(output): %v\noutput: %s", err, out)
	}
	target := findByID(root, id)
	if target == nil {
		t.Fatalf("no element with id %q in output: %s", id, out)
	}
	return getAttr(target, attr)
}

const translateSVG = `<svg viewBox="0 0 100 100">
  <g id="box">
    <animateTransform attributeName="transform" type="translate"
      values="0,0;10,0" dur="1s" repeatCount="indefinite"/>
  </g>
</svg>`

func TestEvaluateLinearTranslate(t *testing.T) {
	epoch := time.Unix(0, 0)
	cases := []struct {
		at   time.Time
		want string
	}{
		{epoch, "translate(0,0)"},
		{epoch.Add(500 * time.Millisecond), "translate(5,0)"},
		{epoch.Add(1 * time.Second), "translate(0,0)"}, // wraps at the loop boundary
		{epoch.Add(2500 * time.Millisecond), "translate(5,0)"},
	}
	for _, tt := range cases {
		got := evalAttr(t, []byte(translateSVG), tt.at, "box", "transform")
		if got != tt.want {
			t.Errorf("at %v: transform = %q, want %q", tt.at.Sub(epoch), got, tt.want)
		}
	}
}

const blinkSVG = `<svg viewBox="0 0 100 100">
  <g id="eyes">
    <animate attributeName="opacity" calcMode="discrete"
      values="1;1;0;1" keyTimes="0;0.85;0.9;1" dur="3s" repeatCount="indefinite"/>
  </g>
</svg>`

func TestEvaluateDiscreteBlink(t *testing.T) {
	epoch := time.Unix(0, 0)
	cases := []struct {
		at   time.Duration
		want string
	}{
		{0, "1"},
		{1000 * time.Millisecond, "1"},
		{2800 * time.Millisecond, "0"}, // inside the [0.9, 1.0) closed segment
		{2999 * time.Millisecond, "0"},
	}
	for _, tt := range cases {
		got := evalAttr(t, []byte(blinkSVG), epoch.Add(tt.at), "eyes", "opacity")
		if got != tt.want {
			t.Errorf("at %v: opacity = %q, want %q", tt.at, got, tt.want)
		}
	}
}

const composeSVG = `<svg viewBox="0 0 100 100">
  <g id="arm" transform="translate(50,50)">
    <animateTransform attributeName="transform" type="rotate"
      values="-10 0 0;10 0 0" dur="1s" repeatCount="indefinite"/>
  </g>
</svg>`

func TestEvaluateComposesWithExistingTransform(t *testing.T) {
	got := evalAttr(t, []byte(composeSVG), time.Unix(0, 0).Add(500*time.Millisecond), "arm", "transform")
	want := "translate(50,50) rotate(0,0,0)"
	if got != want {
		t.Errorf("transform = %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, "translate(50,50) ") {
		t.Errorf("baked transform must preserve the static anchor: %q", got)
	}
}

const repeatCountSVG = `<svg viewBox="0 0 100 100">
  <g id="box">
    <animateTransform attributeName="transform" type="translate"
      values="0,0;10,0" dur="1s" repeatCount="2"/>
  </g>
</svg>`

func TestEvaluateFiniteRepeatCountFreezesAtEnd(t *testing.T) {
	epoch := time.Unix(0, 0)
	if got := evalAttr(t, []byte(repeatCountSVG), epoch.Add(1500*time.Millisecond), "box", "transform"); got != "translate(5,0)" {
		t.Errorf("mid second repeat: transform = %q, want translate(5,0)", got)
	}
	if got := evalAttr(t, []byte(repeatCountSVG), epoch.Add(10*time.Second), "box", "transform"); got != "translate(10,0)" {
		t.Errorf("after repeats exhausted: transform = %q, want frozen at last keyframe translate(10,0)", got)
	}
}

func TestEvaluateStripsAnimationElementsFromOutput(t *testing.T) {
	out, err := Evaluate([]byte(translateSVG), time.Unix(0, 0))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if strings.Contains(string(out), "animateTransform") {
		t.Errorf("baked output still contains an animateTransform element: %s", out)
	}
}

func TestEvaluateLeavesUnsupportedAnimationInPlace(t *testing.T) {
	const missingValues = `<svg viewBox="0 0 100 100">
  <g id="box">
    <animateTransform attributeName="transform" type="translate" dur="1s" repeatCount="indefinite"/>
  </g>
</svg>`
	out, err := Evaluate([]byte(missingValues), time.Unix(0, 0))
	if err != nil {
		t.Fatalf("Evaluate should not error on an unsupported animation, got: %v", err)
	}
	root, err := parseSVG(out)
	if err != nil {
		t.Fatalf("output must still be valid SVG: %v", err)
	}
	if findByID(root, "box") == nil {
		t.Fatal("the box group should still be present")
	}
}

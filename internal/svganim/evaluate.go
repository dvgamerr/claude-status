// Package svganim evaluates a small, deliberately limited subset of SMIL
// (<animate>/<animateTransform>/<set>) against a point in time and bakes the
// result into a plain static SVG. It exists because github.com/srwiley/oksvg
// only rasterizes static SVGs and has no animation support at all — this
// package is the missing "what does the animation look like at time T" step
// that runs before handing the SVG to oksvg.
package svganim

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// node is a minimal DOM: enough structure to find <animate*> children,
// bake their computed value onto the parent, strip them, and re-serialize
// valid SVG. Namespaces are deliberately ignored (dropped to the local
// name) since none of this project's SVGs use prefixed elements/attributes
// beyond the root's default xmlns.
type node struct {
	name     string
	attrs    []xml.Attr
	children []*node
	text     string
}

// Evaluate parses source as SVG, bakes every <animate>/<animateTransform>/
// <set> element it finds into a static attribute on its parent for the
// instant now, strips the animation elements, and returns the resulting
// plain SVG. Elements whose animation attributes aren't recognized are left
// in place unmodified rather than erroring, so one malformed group degrades
// gracefully instead of failing the whole render.
func Evaluate(source []byte, now time.Time) ([]byte, error) {
	root, err := parseSVG(source)
	if err != nil {
		return nil, err
	}
	evaluateNode(root, now)
	var buf bytes.Buffer
	writeNode(&buf, root)
	return buf.Bytes(), nil
}

func parseSVG(source []byte) (*node, error) {
	dec := xml.NewDecoder(bytes.NewReader(source))
	var stack []*node
	var root *node
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{name: t.Name.Local, attrs: append([]xml.Attr(nil), t.Attr...)}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, n)
			} else {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text += string(t)
			}
		}
	}
	if root == nil {
		return nil, errors.New("svganim: no root element")
	}
	return root, nil
}

// evaluateNode bakes and strips this node's own <animate*> children, then
// recurses into whatever's left.
func evaluateNode(n *node, now time.Time) {
	content := n.children[:0:0]
	for _, child := range n.children {
		switch child.name {
		case "animate", "animateTransform":
			if attrName, value, ok := evaluateAnimate(child, now); ok {
				applyAttr(n, attrName, value)
			}
		case "set":
			if attrName := getAttr(child, "attributeName"); attrName != "" {
				applyAttr(n, attrName, getAttr(child, "to"))
			}
		default:
			content = append(content, child)
		}
	}
	n.children = content
	for _, c := range n.children {
		evaluateNode(c, now)
	}
}

// applyAttr sets attribute name to value on n. A "transform" value is
// composed after any existing static transform (space-separated) so an
// authored anchor transform establishes the pivot and the baked value
// animates relative to it; every other attribute is simply overwritten.
func applyAttr(n *node, name, value string) {
	for i, a := range n.attrs {
		if a.Name.Local == name {
			if name == "transform" && a.Value != "" {
				n.attrs[i].Value = a.Value + " " + value
			} else {
				n.attrs[i].Value = value
			}
			return
		}
	}
	n.attrs = append(n.attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

// evaluateAnimate computes the baked attribute name/value for an <animate>
// or <animateTransform> element at time now. Only the values-list form is
// supported (no from/to/by) since that's all this project's rigs use.
func evaluateAnimate(el *node, now time.Time) (attrName, value string, ok bool) {
	attrName = getAttr(el, "attributeName")
	if attrName == "" {
		return "", "", false
	}
	dur, err := parseClockValue(getAttr(el, "dur"))
	if err != nil || dur <= 0 {
		return "", "", false
	}
	keyframes, err := parseKeyframes(getAttr(el, "values"))
	if err != nil || len(keyframes) < 2 {
		return "", "", false
	}
	keyTimes, err := parseKeyTimes(getAttr(el, "keyTimes"), len(keyframes))
	if err != nil {
		return "", "", false
	}

	begin, _ := parseClockValue(getAttr(el, "begin")) // defaults to 0 on error
	phase := loopPhase(now, dur, begin, getAttr(el, "repeatCount"))

	i := 0
	for i < len(keyTimes)-2 && phase > keyTimes[i+1] {
		i++
	}
	span := keyTimes[i+1] - keyTimes[i]
	u := 0.0
	if span > 0 {
		u = (phase - keyTimes[i]) / span
	}
	u = clamp01(u)

	calcMode := getAttr(el, "calcMode")
	out := interpolate(keyframes[i], keyframes[i+1], u, calcMode == "discrete")
	if len(out) == 0 {
		return "", "", false
	}

	if attrName != "transform" {
		return attrName, formatFloat(out[0]), true
	}
	baked, ok := formatTransform(getAttr(el, "type"), out)
	return attrName, baked, ok
}

// loopPhase returns the elapsed position within [0, dur] for an animation
// that began at begin and repeats per repeatCount ("indefinite", a numeric
// repeat count, or empty which this package also treats as indefinite since
// every animation this project authors loops forever).
func loopPhase(now time.Time, dur, begin time.Duration, repeatCount string) float64 {
	elapsed := now.UnixMilli() - begin.Milliseconds()
	durMS := dur.Milliseconds()

	indefinite := repeatCount == "" || repeatCount == "indefinite"
	var phaseMS int64
	if indefinite {
		phaseMS = ((elapsed % durMS) + durMS) % durMS
	} else if repeatN, err := strconv.ParseFloat(repeatCount, 64); err == nil {
		total := int64(float64(durMS) * repeatN)
		switch {
		case elapsed < 0:
			phaseMS = 0
		case elapsed >= total:
			phaseMS = durMS
		default:
			phaseMS = elapsed % durMS
		}
	} else {
		phaseMS = ((elapsed % durMS) + durMS) % durMS
	}
	return float64(phaseMS) / float64(durMS)
}

func interpolate(a, b []float64, u float64, discrete bool) []float64 {
	if discrete {
		return a
	}
	n := min(len(b), len(a))
	out := make([]float64, n)
	for i := range n {
		out[i] = a[i] + (b[i]-a[i])*u
	}
	return out
}

func formatTransform(typ string, v []float64) (string, bool) {
	switch typ {
	case "translate":
		y := 0.0
		if len(v) > 1 {
			y = v[1]
		}
		return fmt.Sprintf("translate(%s,%s)", formatFloat(v[0]), formatFloat(y)), true
	case "rotate":
		if len(v) >= 3 {
			return fmt.Sprintf("rotate(%s,%s,%s)", formatFloat(v[0]), formatFloat(v[1]), formatFloat(v[2])), true
		}
		return fmt.Sprintf("rotate(%s)", formatFloat(v[0])), true
	case "scale":
		if len(v) >= 2 {
			return fmt.Sprintf("scale(%s,%s)", formatFloat(v[0]), formatFloat(v[1])), true
		}
		return fmt.Sprintf("scale(%s)", formatFloat(v[0])), true
	default:
		return "", false
	}
}

func clamp01(u float64) float64 {
	if u < 0 {
		return 0
	}
	if u > 1 {
		return 1
	}
	return u
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// parseClockValue parses a SMIL clock value: "600ms", "0.6s", or a bare
// number (treated as seconds).
func parseClockValue(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("svganim: empty clock value")
	}
	switch {
	case strings.HasSuffix(s, "ms"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "ms"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(v * float64(time.Millisecond)), nil
	case strings.HasSuffix(s, "s"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "s"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(v * float64(time.Second)), nil
	default:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("svganim: invalid clock value %q", s)
		}
		return time.Duration(v * float64(time.Second)), nil
	}
}

// parseKeyframes splits a values="a,b;c,d;..." attribute into one float
// slice per ";"-separated keyframe.
func parseKeyframes(values string) ([][]float64, error) {
	if values == "" {
		return nil, errors.New("svganim: empty values")
	}
	parts := strings.Split(values, ";")
	out := make([][]float64, 0, len(parts))
	for _, part := range parts {
		fs, err := parseFloatList(part)
		if err != nil {
			return nil, err
		}
		out = append(out, fs)
	}
	return out, nil
}

// parseKeyTimes parses a keyTimes="0;0.5;1" attribute, or defaults to n
// evenly spaced keyframes over [0,1] when absent.
func parseKeyTimes(keyTimes string, n int) ([]float64, error) {
	if keyTimes == "" {
		out := make([]float64, n)
		for i := range out {
			out[i] = float64(i) / float64(n-1)
		}
		return out, nil
	}
	out, err := parseFloatList(strings.ReplaceAll(keyTimes, ";", " "))
	if err != nil {
		return nil, err
	}
	if len(out) != n {
		return nil, fmt.Errorf("svganim: keyTimes has %d entries, want %d", len(out), n)
	}
	return out, nil
}

func parseFloatList(s string) ([]float64, error) {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	if len(fields) == 0 {
		return nil, fmt.Errorf("svganim: empty number list %q", s)
	}
	out := make([]float64, len(fields))
	for i, f := range fields {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil, fmt.Errorf("svganim: invalid number %q: %w", f, err)
		}
		out[i] = v
	}
	return out, nil
}

func getAttr(n *node, key string) string {
	for _, a := range n.attrs {
		if a.Name.Local == key {
			return a.Value
		}
	}
	return ""
}

func writeNode(buf *bytes.Buffer, n *node) {
	buf.WriteByte('<')
	buf.WriteString(n.name)
	for _, a := range n.attrs {
		buf.WriteByte(' ')
		buf.WriteString(attrName(a.Name))
		buf.WriteString(`="`)
		buf.WriteString(escapeAttr(a.Value))
		buf.WriteByte('"')
	}
	hasText := strings.TrimSpace(n.text) != ""
	if len(n.children) == 0 && !hasText {
		buf.WriteString("/>")
		return
	}
	buf.WriteByte('>')
	if hasText {
		buf.WriteString(escapeText(n.text))
	}
	for _, c := range n.children {
		writeNode(buf, c)
	}
	buf.WriteString("</")
	buf.WriteString(n.name)
	buf.WriteByte('>')
}

func attrName(name xml.Name) string {
	switch name.Space {
	case "":
		return name.Local
	case "xmlns":
		return "xmlns:" + name.Local
	default:
		return name.Space + ":" + name.Local
	}
}

func escapeText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func escapeAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// ahemFonts measures as Chrome draws the Ahem font: every glyph a square of the font size,
// ascent 0.8 em, descent 0.2 em, and zero-width spaces taking no room.
type ahemFonts struct{}

func (ahemFonts) Metrics(f style.Font) box.Metrics {
	return box.Metrics{Ascent: 0.8 * f.Size, Descent: 0.2 * f.Size}
}

func (ahemFonts) Advance(f style.Font, text string) float32 {
	n := utf8.RuneCountInString(strings.ReplaceAll(text, ahemBreak, ""))
	return float32(n) * f.Size
}

// unsupported are the fixture values ui's style can't say yet: alignment safety,
// self-start/self-end, grid line names and stretch sizes.
func unsupported(f taffyFixture) string {
	reason := ""
	f.Root.walk(func(n *fixtureNode) {
		for k, v := range n.Style {
			switch {
			case strings.HasPrefix(v, "safe") || strings.HasPrefix(v, "unsafe"):
				reason = "alignment safety"
			case strings.HasPrefix(v, "self-"):
				reason = "self-start/self-end"
			case strings.Contains(v, "["):
				reason = "grid line names"
			case strings.HasPrefix(k, "grid-") && strings.IndexFunc(v, func(r rune) bool { return r >= 'a' && r <= 'z' }) >= 0 &&
				!strings.HasPrefix(v, "span") && !strings.Contains(v, "px") && !strings.Contains(v, "fr") &&
				!strings.Contains(v, "content") && !strings.Contains(v, "auto") && !strings.Contains(v, "minmax") &&
				!strings.Contains(v, "repeat") && !strings.Contains(v, "%") && k != "grid-auto-flow":
				reason = "grid line names"
			case v == "stretch" && (k == "width" || k == "height" || k == "flex-basis"):
				reason = "stretch sizes"
			case k == "scrollbar-width" && strings.Contains(v, "px") == false && v != "0" && v != "15":
				reason = "scrollbar width " + v
			}
		}
	})
	return reason
}

func TestTaffy(t *testing.T) {
	only := os.Getenv("TAFFY")
	for _, suite := range taffySuites {
		fixtures, err := loadTaffySuite(suite)
		if err != nil {
			t.Fatal(err)
		}
		pass, total := 0, 0
		var failed []string
		for _, f := range fixtures {
			if only != "" && !strings.Contains(f.Name, only) {
				continue
			}
			if unsupported(f) != "" {
				continue
			}
			total++
			if msg := runFixture(f); msg != "" {
				failed = append(failed, f.Name+": "+msg)
				continue
			}
			pass++
		}
		t.Logf("%s: %d/%d pass (%d fixtures)", suite, pass, total, len(fixtures))
		if os.Getenv("TAFFY_VERBOSE") != "" {
			for _, m := range failed {
				t.Log("  " + m)
			}
		}
	}
}

func runFixture(f taffyFixture) (msg string) { //nolint:nonamedreturns // the deferred recover sets msg
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint("panic: ", r)
		}
	}()
	root, nodes, err := fixtureDOM(f.Root, true)
	if err != nil {
		return err.Error()
	}
	avail := spaces{viewportSpace(f.Viewport[0]), viewportSpace(f.Viewport[1])}
	if avail.w.kind != maxContent || avail.h.kind != maxContent {
		root = viewportDiv(root, avail)
		avail = maxContentSpaces
	}
	tr := layoutIn(root, avail, ahemFonts{}, f.Rounding)
	if os.Getenv("TAFFY_DUMP") != "" {
		dumpBoxes(tr.Root, "")
	}
	// Expected positions are offsetLeft/offsetTop: from the nearest positioned ancestor.
	var diffs []string
	var walk func(n *fixtureNode, path string, offsetParent *box.Box, hidden bool)
	walk = func(n *fixtureNode, path string, offsetParent *box.Box, hidden bool) {
		b := tr.Of[nodes[n]]
		if b == nil {
			diffs = append(diffs, path+": no box")
			return
		}
		hidden = hidden || n.Style["display"] == "none"
		got := fixtureBox{X: b.Pos.X, Y: b.Pos.Y, Width: b.Size.X, Height: b.Size.Y}
		if !hidden && offsetParent != nil {
			r, o := b.Rect(), offsetParent.Rect()
			got.X, got.Y = r.X-o.X, r.Y-o.Y
		}
		if !near(got.X, n.Want.X) || !near(got.Y, n.Want.Y) || !near(got.Width,
			n.Want.Width) || !near(got.Height, n.Want.Height) {
			diffs = append(diffs,
				fmt.Sprintf("%s: got %v,%v %vx%v want %v,%v %vx%v", path, got.X, got.Y,
					got.Width, got.Height, n.Want.X, n.Want.Y, n.Want.Width, n.Want.Height))
		}
		next := offsetParent
		if offsetParent == nil || nodes[n] == nil || dom.Data(nodes[n]).Style.Position != style.Static {
			next = b
		}
		for i, c := range n.Children {
			walk(c, fmt.Sprintf("%s/%d", path, i), next, hidden)
		}
	}
	walk(f.Root, "root", nil, false)
	if len(diffs) > 0 {
		if len(diffs) > 3 {
			diffs = append(diffs[:3], fmt.Sprintf("… %d more", len(diffs)-3))
		}
		return strings.Join(diffs, "; ")
	}
	return ""
}

func near(a, b float32) bool { return a-b < 0.1 && b-a < 0.1 }

func dumpBoxes(b *box.Box, indent string) {
	fmt.Fprintf(os.Stderr, "%s%v %v,%v %vx%v %d\n", indent, b.Kind, b.Pos.X, b.Pos.Y, b.Size.X, b.Size.Y, len(b.Lines))
	for _, c := range b.Children {
		dumpBoxes(c, indent+"  ")
	}
}

// viewportDiv is gentest's .viewport around a test root with a viewport: an absolutely
// positioned flex container aligning its item at the start, sized by the viewport.
func viewportDiv(root *dom.Node, avail spaces) *dom.Node {
	v := dom.NewElement(dom.Div)
	st := style.Initial()
	st.Display = style.FlexDisplay
	st.Position = style.Absolute
	st.AlignItems = style.AlignStart
	st.JustifyContent = style.AlignStart
	st.Size = style.Size[style.Length]{Width: style.Length{Kind: style.MaxContent},
		Height: style.Length{Kind: style.MaxContent}}
	if avail.w.definite() {
		st.Size.Width = style.Px(avail.w.v)
	}
	if avail.h.definite() {
		st.Size.Height = style.Px(avail.h.v)
	}
	dom.Data(v).Style = &st
	v.Append(root)
	return v
}

func viewportSpace(v string) space {
	switch v {
	case "max-content", "":
		return maxContentSpace
	case "min-content":
		return minContentSpace
	}
	f, err := strconv.ParseFloat(strings.TrimSuffix(v, "px"), 32)
	if err != nil {
		return maxContentSpace
	}
	return def(float32(f))
}

// fixtureDOM builds the fixture's tree as ui nodes with computed styles, under the
// gentest stylesheet: divs are flex, relative and border-box, text is 10px Ahem with a line
// height of 1, and the root is absolutely positioned.
func fixtureDOM(fn *fixtureNode, isRoot bool) (*dom.Node, map[*fixtureNode]*dom.Node, error) {
	nodes := map[*fixtureNode]*dom.Node{}
	var build func(fn *fixtureNode, isRoot bool) (*dom.Node, error)
	build = func(fn *fixtureNode, isRoot bool) (*dom.Node, error) {
		e := dom.NewElement(dom.Div)
		st := style.Initial()
		st.Display = style.FlexDisplay
		if isRoot {
			st.Position = style.Absolute
		}
		st.Font = style.Font{Family: []string{"ahem"}, Size: 10, Weight: 400}
		st.LineHeight = style.LineHeight{Factor: 1}
		for k, v := range fn.Style {
			if err := applyFixtureStyle(&st, k, v); err != nil {
				return nil, fmt.Errorf("%s=%q: %w", k, v, err)
			}
		}
		dom.Data(e).Style = &st
		nodes[fn] = e
		if fn.IsText {
			txt := dom.NewText(fn.Text)
			ts := st
			dom.Data(txt).Style = &ts
			e.Append(txt)
		}
		for _, c := range fn.Children {
			ce, err := build(c, false)
			if err != nil {
				return nil, err
			}
			e.Append(ce)
		}
		return e, nil
	}
	root, err := build(fn, isRoot)
	return root, nodes, err
}

func parseLength(v string) (style.Length, error) {
	v = strings.TrimSpace(v)
	switch v {
	case "auto":
		return style.Length{}, nil
	case "min-content":
		return style.Length{Kind: style.MinContent}, nil
	case "max-content":
		return style.Length{Kind: style.MaxContent}, nil
	case "fit-content":
		return style.Length{Kind: style.FitContent}, nil
	case "content":
		return style.Length{Kind: style.MaxContent}, nil
	case "none":
		return style.Length{Kind: style.None}, nil
	}
	if inner, ok := strings.CutPrefix(v, "fit-content("); ok {
		l, err := parseLength(strings.TrimSuffix(inner, ")"))
		l.Kind = style.FitContent
		return l, err
	}
	if s, ok := strings.CutSuffix(v, "fr"); ok {
		f, err := strconv.ParseFloat(s, 32)
		return style.Length{Kind: style.Fr, Px: float32(f)}, err
	}
	if s, ok := strings.CutSuffix(v, "%"); ok {
		f, err := strconv.ParseFloat(s, 32)
		l := style.Percent(float32(f))
		l.IsPercent = true
		return l, err
	}
	f, err := strconv.ParseFloat(strings.TrimSuffix(v, "px"), 32)
	return style.Px(float32(f)), err
}

func parseNumber(v string) (float32, error) {
	f, err := strconv.ParseFloat(strings.TrimSuffix(v, "px"), 32)
	return float32(f), err
}

var aligns = map[string]style.Align{ //nolint:gochecknoglobals // a read-only test table
	"normal": style.AlignNormal, "auto": style.AlignAuto, "stretch": style.AlignStretch,
	"start": style.AlignStart, "end": style.AlignEnd, "flex-start": style.AlignFlexStart,
	"flex-end": style.AlignFlexEnd, "center": style.AlignCenter, "baseline": style.AlignBaseline,
	"space-between": style.AlignSpaceBetween, "space-around": style.AlignSpaceAround,
	"space-evenly": style.AlignSpaceEvenly, "left": style.AlignStart, "right": style.AlignEnd,
}

func parseAlign(v string) (style.Align, error) {
	a, ok := aligns[v]
	if !ok {
		return 0, fmt.Errorf("unknown alignment")
	}
	return a, nil
}

//nolint:cyclop,gocyclo // one case per fixture attribute
func applyFixtureStyle(st *style.Computed, k, v string) error {
	var err error
	length := func(dst *style.Length) {
		*dst, err = parseLength(v)
	}
	number := func(dst *float32) {
		*dst, err = parseNumber(v)
	}
	alignment := func(dst *style.Align) {
		*dst, err = parseAlign(v)
	}
	overflow := func(dst *style.Overflow) {
		switch v {
		case "visible":
			*dst = style.Visible
		case "hidden":
			*dst = style.Hidden
		case "clip":
			*dst = style.Clip
		case "scroll":
			*dst = style.Scroll
		case "auto":
			*dst = style.OverflowAuto
		default:
			err = fmt.Errorf("unknown overflow")
		}
	}
	switch k {
	case "display":
		switch v {
		case "flex":
			st.Display = style.FlexDisplay
		case "grid":
			st.Display = style.GridDisplay
		case "block":
			st.Display = style.Block
		case "none":
			st.Display = style.DisplayNone
		default:
			err = fmt.Errorf("unknown display")
		}
	case "position":
		switch v {
		case "absolute":
			st.Position = style.Absolute
		case "relative":
			st.Position = style.Relative
		case "static":
			st.Position = style.Static
		case "fixed":
			st.Position = style.FixedPosition
		case "sticky":
			st.Position = style.Sticky
		}
	case "box-sizing":
		if v == "content-box" {
			st.BoxSizing = style.ContentBox
		}
	case "width":
		length(&st.Size.Width)
	case "height":
		length(&st.Size.Height)
	case "min-width":
		length(&st.MinSize.Width)
	case "min-height":
		length(&st.MinSize.Height)
	case "max-width":
		length(&st.MaxSize.Width)
	case "max-height":
		length(&st.MaxSize.Height)
	case "aspect-ratio":
		number(&st.AspectRatio)
	case "top":
		length(&st.Inset.Top)
	case "right":
		length(&st.Inset.Right)
	case "bottom":
		length(&st.Inset.Bottom)
	case "left":
		length(&st.Inset.Left)
	case "margin-top":
		length(&st.Margin.Top)
	case "margin-right":
		length(&st.Margin.Right)
	case "margin-bottom":
		length(&st.Margin.Bottom)
	case "margin-left":
		length(&st.Margin.Left)
	case "padding-top":
		length(&st.Padding.Top)
	case "padding-right":
		length(&st.Padding.Right)
	case "padding-bottom":
		length(&st.Padding.Bottom)
	case "padding-left":
		length(&st.Padding.Left)
	case "border-top":
		number(&st.Border.Top)
	case "border-right":
		number(&st.Border.Right)
	case "border-bottom":
		number(&st.Border.Bottom)
	case "border-left":
		number(&st.Border.Left)
	case "flex-direction":
		switch v {
		case "row":
			st.Direction = style.Row
		case "row-reverse":
			st.Direction = style.RowReverse
		case "column":
			st.Direction = style.Column
		case "column-reverse":
			st.Direction = style.ColumnReverse
		}
	case "flex-wrap":
		switch v {
		case "wrap":
			st.Wrap = style.Wrap
		case "wrap-reverse":
			st.Wrap = style.WrapReverse
		case "nowrap":
			st.Wrap = style.NoWrap
		}
	case "flex-grow":
		number(&st.Grow)
	case "flex-shrink":
		number(&st.Shrink)
	case "flex-basis":
		length(&st.Basis)
	case "column-gap":
		length(&st.Gap.Width)
	case "row-gap":
		length(&st.Gap.Height)
	case "align-items":
		alignment(&st.AlignItems)
	case "align-self":
		alignment(&st.AlignSelf)
	case "align-content":
		alignment(&st.AlignContent)
	case "justify-content":
		alignment(&st.JustifyContent)
	case "justify-items":
		alignment(&st.JustifyItems)
	case "justify-self":
		alignment(&st.JustifySelf)
	case "overflow-x":
		overflow(&st.Overflow.Width)
	case "overflow-y":
		overflow(&st.Overflow.Height)
	case "scrollbar-width":
		number(&st.ScrollbarWidth)
	case "grid-template-columns":
		st.Columns, err = parseTrackList(v)
	case "grid-template-rows":
		st.Rows, err = parseTrackList(v)
	case "grid-auto-columns":
		st.AutoColumns, err = parseTracks(v)
	case "grid-auto-rows":
		st.AutoRows, err = parseTracks(v)
	case "grid-auto-flow":
		switch v {
		case "row":
			st.Flow = style.FlowRow
		case "column":
			st.Flow = style.FlowColumn
		case "row dense", "dense":
			st.Flow = style.FlowRowDense
		case "column dense":
			st.Flow = style.FlowColumnDense
		}
	case "grid-column-start":
		st.Column[0], err = parseGridLine(v)
	case "grid-column-end":
		st.Column[1], err = parseGridLine(v)
	case "grid-row-start":
		st.Row[0], err = parseGridLine(v)
	case "grid-row-end":
		st.Row[1], err = parseGridLine(v)
	default:
		err = fmt.Errorf("unmapped")
	}
	return err
}

func parseGridLine(v string) (style.GridLine, error) {
	if v == "auto" {
		return style.GridLine{}, nil
	}
	if s, ok := strings.CutPrefix(v, "span "); ok {
		n, err := strconv.Atoi(s)
		return style.GridLine{Span: n}, err
	}
	n, err := strconv.Atoi(v)
	return style.GridLine{Line: n}, err
}

// splitTopLevel splits on spaces outside parentheses.
func splitTopLevel(v string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range v {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ' ':
			if depth == 0 {
				if i > start {
					out = append(out, v[start:i])
				}
				start = i + 1
			}
		}
	}
	if start < len(v) {
		out = append(out, v[start:])
	}
	return out
}

func parseTrack(v string) (style.Track, error) {
	if inner, ok := strings.CutPrefix(v, "minmax("); ok {
		parts := strings.SplitN(strings.TrimSuffix(inner, ")"), ",", 2)
		if len(parts) != 2 {
			return style.Track{}, fmt.Errorf("bad minmax")
		}
		lo, err := parseLength(parts[0])
		if err != nil {
			return style.Track{}, err
		}
		hi, err := parseLength(parts[1])
		return style.Track{Min: lo, Max: hi}, err
	}
	l, err := parseLength(v)
	return style.Track{Min: l, Max: l}, err
}

func parseTracks(v string) ([]style.Track, error) {
	var out []style.Track
	for _, p := range splitTopLevel(v) {
		tr, err := parseTrack(p)
		if err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, nil
}

func parseTrackList(v string) ([]style.TrackRepeat, error) {
	if v == "none" {
		return nil, nil
	}
	var out []style.TrackRepeat
	for _, p := range splitTopLevel(v) {
		if inner, ok := strings.CutPrefix(p, "repeat("); ok {
			inner = strings.TrimSuffix(inner, ")")
			count, rest, _ := strings.Cut(inner, ",")
			tracks, err := parseTracks(strings.TrimSpace(rest))
			if err != nil {
				return nil, err
			}
			r := style.TrackRepeat{Tracks: tracks}
			switch strings.TrimSpace(count) {
			case "auto-fill":
				r.Count = style.AutoFill
			case "auto-fit":
				r.Count = style.AutoFit
			default:
				r.Count, err = strconv.Atoi(strings.TrimSpace(count))
				if err != nil {
					return nil, err
				}
			}
			out = append(out, r)
			continue
		}
		tr, err := parseTrack(p)
		if err != nil {
			return nil, err
		}
		out = append(out, style.TrackRepeat{Count: 1, Tracks: []style.Track{tr}})
	}
	return out, nil
}

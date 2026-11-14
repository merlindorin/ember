//go:build chrome

package css_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// compared lists the properties checked against Chrome, each with how to
// write ours as getComputedStyle does.
//
//nolint:gochecknoglobals // the properties compared, a test table
var compared = map[string]func(s *style.Computed) string{
	"color":              func(s *style.Computed) string { return rgb(s.Color) },
	"background-color":   func(s *style.Computed) string { return rgb(s.Background) },
	"border-top-color":   func(s *style.Computed) string { return rgb(s.BorderColor.Top) },
	"border-left-color":  func(s *style.Computed) string { return rgb(s.BorderColor.Left) },
	"border-top-width":   func(s *style.Computed) string { return px(s.Border.Top) },
	"border-right-width": func(s *style.Computed) string { return px(s.Border.Right) },
	"border-bottom-style": func(s *style.Computed) string {
		return []string{"none", "solid", "dashed", "dotted", "inset", "outset"}[s.BorderStyle.Bottom]
	},
	"font-size":   func(s *style.Computed) string { return px(s.Font.Size) },
	"font-weight": func(s *style.Computed) string { return strconv.Itoa(s.Font.Weight) },
	"font-style": func(s *style.Computed) string {
		return map[bool]string{true: "italic", false: "normal"}[s.Font.Italic]
	},
	"font-family": func(s *style.Computed) string { return strings.Join(s.Font.Family, ", ") },
	"line-height": func(s *style.Computed) string {
		switch {
		case s.LineHeight.Normal:
			return "normal"
		case s.LineHeight.Factor != 0:
			return px(s.LineHeight.Factor * s.Font.Size)
		}
		return px(s.LineHeight.Px)
	},
	"letter-spacing": func(s *style.Computed) string { return px(s.LetterSpacing) },
	"display": func(s *style.Computed) string {
		if s.ListItem {
			return "list-item"
		}
		return []string{"inline", "block", "inline-block", "flex", "inline-flex", "grid", "inline-grid", "none",
			"contents"}[s.Display]
	},
	"position": func(s *style.Computed) string {
		return []string{"static", "relative", "absolute", "fixed", "sticky"}[s.Position]
	},
	"padding-top":  func(s *style.Computed) string { return length(s.Padding.Top) },
	"padding-left": func(s *style.Computed) string { return length(s.Padding.Left) },
	"margin-top":   func(s *style.Computed) string { return length(s.Margin.Top) },
	"margin-right": func(s *style.Computed) string { return length(s.Margin.Right) },
	"opacity":      func(s *style.Computed) string { return num(s.Opacity) },
	"text-align": func(s *style.Computed) string {
		return []string{"start", "end", "left", "right", "center", "justify"}[s.TextAlign]
	},
	"text-transform": func(s *style.Computed) string {
		return []string{"none", "uppercase", "lowercase", "capitalize"}[s.TextTransform]
	},
	"white-space": func(s *style.Computed) string {
		return []string{"normal", "nowrap", "pre", "pre-wrap", "pre-line"}[s.WhiteSpace]
	},
	"overflow-x": func(s *style.Computed) string { return overflow(s.Overflow.Width) },
	"overflow-y": func(s *style.Computed) string { return overflow(s.Overflow.Height) },
	"z-index": func(s *style.Computed) string {
		return map[bool]string{true: "auto", false: strconv.Itoa(s.ZIndex)}[s.ZAuto]
	},
	"flex-grow":   func(s *style.Computed) string { return num(s.Grow) },
	"flex-shrink": func(s *style.Computed) string { return num(s.Shrink) },
	"order":       func(s *style.Computed) string { return strconv.Itoa(s.Order) },
	"visibility": func(s *style.Computed) string {
		return map[bool]string{true: "visible", false: "hidden"}[s.Visible]
	},
	"pointer-events": func(s *style.Computed) string {
		return map[bool]string{true: "none", false: "auto"}[s.PointerNone]
	},
	"text-overflow": func(s *style.Computed) string {
		return map[bool]string{true: "ellipsis", false: "clip"}[s.Ellipsis]
	},
	"column-gap": func(s *style.Computed) string {
		if s.Gap.Width.Kind == style.Auto {
			return "normal"
		}
		return length(s.Gap.Width)
	},
	"outline-width":  func(s *style.Computed) string { return px(s.OutlineWidth) },
	"outline-offset": func(s *style.Computed) string { return px(s.OutlineOffset) },
}

func rgb(c color.Color) string {
	s := c.SRGB()
	ch := func(v float32) int { return int(math.Round(float64(v) * 255)) }
	if c.A >= 1 {
		return fmt.Sprintf("rgb(%d, %d, %d)", ch(s[0]), ch(s[1]), ch(s[2]))
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %s)", ch(s[0]), ch(s[1]), ch(s[2]), num(c.A))
}

func px(v float32) string { return num(v) + "px" }

func num(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) }

func length(l style.Length) string {
	if l.Kind == style.Auto {
		return "auto"
	}
	return px(l.Px)
}

func overflow(o style.Overflow) string {
	return []string{"visible", "hidden", "clip", "scroll", "auto"}[o]
}

// same compares two computed values: numbers within 0.01 wherever they
// stand, colours channel by channel within one step.
func same(prop, ours, chrome string) bool {
	norm := func(s string) string { return strings.ReplaceAll(s, `"`, "") }
	ours, chrome = norm(ours), norm(pageURL.ReplaceAllStringFunc(oklchToRGB(chrome), func(m string) string {
		p, _ := url.PathUnescape(pageURL.FindStringSubmatch(m)[1])
		return "url(" + p + ")"
	}))
	if ours == chrome {
		return true
	}
	switch {
	case (prop == "min-width" || prop == "min-height") && ours == "auto" && chrome == "0px":
		// Chrome reports min-*: auto as 0px on a box that is no flex or
		// grid item, and on no box at all.
		return true
	case prop == "transition-property" && ours == "all" && chrome == "none":
		// none and the initial all 0s both mean no transition: Transitions
		// is empty for both.
		return true
	case prop == "aspect-ratio" && strings.Contains(chrome, " / "):
		var w, h float64
		if _, err := fmt.Sscanf(chrome, "%g / %g", &w, &h); err != nil {
			return false
		}
		r, err := strconv.ParseFloat(ours, 64)
		return err == nil && h != 0 && math.Abs(r-w/h) < 1e-4
	}
	if prop == "letter-spacing" && ours == "0px" && chrome == "normal" {
		return true
	}
	if sameNumbers(ours, chrome) {
		return true
	}
	ca, cb := channels(ours), channels(chrome)
	if ca != nil && len(ca) == len(cb) {
		for i := range ca {
			tol := 1.0
			if i == 3 {
				tol = 0.01
			}
			if math.Abs(ca[i]-cb[i]) > tol {
				return false
			}
		}
		return true
	}
	return false
}

func channels(s string) []float64 {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "rgba("), "rgb(")
	if !strings.HasSuffix(s, ")") {
		return nil
	}
	var out []float64
	for _, p := range strings.Split(strings.TrimSuffix(s, ")"), ",") {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil
		}
		out = append(out, v)
	}
	return out
}

func matchChrome(t *testing.T, d *ui.Document) {
	t.Helper()
	matchChromeProps(t, d, compared, 0)
}

// matchChromeProps compares props on every element from the skip-th on.
func matchChromeProps(t *testing.T, d *ui.Document, props map[string]func(*style.Computed) string, skip int) {
	t.Helper()
	d.Layout(geom.V2(800, 600))
	matchChromeShown(t, d, props, skip, "", nil, func(n *dom.Node) *style.Computed { return dom.Data(n).Style })
}

// matchChromeShown compares what read gives with Chrome's computed styles
// after script ran in the page; then, if set, does on ui's side what
// script did in Chrome's.
func matchChromeShown(t *testing.T, d *ui.Document, props map[string]func(*style.Computed) string, skip int,
	script string, then func(), read func(*dom.Node) *style.Computed) {
	t.Helper()
	size := geom.V2(800, 600)
	names := make([]string, 0, len(props))
	for p := range props {
		names = append(names, p)
	}
	slices.Sort(names)
	want, err := chrome.StylesAfter(d, size, names, script)
	if err != nil {
		t.Fatal(err)
	}
	if then != nil {
		then()
	}
	for _, w := range d.Warnings() {
		t.Errorf("warning: %v", w)
	}
	var got []*dom.Node
	d.HTML().Walk(func(n *dom.Node) bool {
		if !n.IsText() {
			got = append(got, n)
		}
		return true
	})
	if len(got) != len(want) {
		t.Fatalf("ember has %d elements, Chrome %d", len(got), len(want))
	}
	checked := 0
	for i, n := range got[skip:] {
		i += skip
		s := read(n)
		for _, p := range names {
			ours := props[p](s)
			if !same(p, ours, want[i][p]) {
				t.Errorf("%s %s: ember %q, Chrome %q", where(n), p, ours, want[i][p])
			}
			checked++
		}
	}
	t.Logf("%d elements, %d values compared", len(got), checked)
}

func where(n *dom.Node) string {
	s := n.Tag().String()
	if n.ID() != "" {
		s += "#" + n.ID()
	}
	return s
}

func TestCascadeMatchesChrome(t *testing.T) {
	fsys := fstest.MapFS{"cascade.css": {Data: []byte(`
:root { --ink: #f7efdd; --pad: 6px; --gap: calc(var(--pad) * 2); }
body { color: var(--ink); font-size: 14px; line-height: 1.5; letter-spacing: 0.05em; }
#spec { color: #00f; }
div.card { color: #0f0; padding: 4px; }
.card.hot { color: #f00; background-color: rgb(10 20 30 / 50%); }
.card { opacity: 0.25; }
.hot { opacity: 0.75; }
div { border-top: 3px solid; }
#imp { margin-top: 5px !important; }
#imp.card { margin-top: 9px; }
.card > p:first-child { font-weight: bold; font-style: italic; }
.card > p:last-child { font-weight: lighter; text-align: center; }
p:not(:first-child, :last-child) { white-space: nowrap; text-overflow: ellipsis; overflow-x: hidden; }
span { font-size: 1.5em; padding-left: 1rem; border-left-color: currentcolor; }
.reset { color: initial; font-size: unset; padding-top: inherit; letter-spacing: normal; }
.flex { display: flex; column-gap: var(--gap); }
.flex > span { flex: 2 0 10px; order: 3; }
.abs { position: absolute; z-index: 4; pointer-events: none; }
.vars {
  --pad: 11px; padding-top: var(--pad); margin-right: var(--missing, 7px);
  outline: 2px solid var(--ink); outline-offset: 1px;
}
.cycle { --a: var(--b); --b: var(--a); color: var(--a, #123456); }
.hidden { visibility: hidden; }
input[type="checkbox"] { background-color: tomato; }
input[type=range][title] { background-color: RebeccaPurple; border-top: 2px solid currentColor; color: darkslategray; }
[title="tip"] { outline-color: currentColor; outline-style: dotted; }
[hidden-ish] { opacity: 0.6; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "cascade.css"))
	d.Body().Append(
		ui.Div(ui.ID("spec"), ui.Class("card", "hot"), ui.Text("id beats classes")),
		ui.Div(ui.Class("card", "hot"), ui.ID("imp"), ui.Style("color: #abc; opacity: 0.5"),
			ui.P(ui.Text("first")),
			ui.P(ui.Text("middle"), ui.Span(ui.Class("reset"), ui.Text("reset"))),
			ui.P(ui.Text("last")),
		),
		ui.Div(ui.Class("flex"),
			ui.Span(ui.Text("item")),
			ui.Span(ui.Class("abs"), ui.Text("abs")),
			ui.Button(ui.Text("button inherits")),
		),
		ui.Div(ui.Class("vars", "cycle"), ui.Span(ui.Class("hidden"), ui.Text("vars"))),
		ui.Div(ui.Style("font-size: 20px; --ink: #0ff"), ui.P(ui.Style("color: var(--ink)"), ui.Text("inline var"))),
		ui.Div(
			ui.Input(ui.Type("checkbox")),
			ui.Input(ui.Type("range"), ui.Attr("title", "tip")),
			ui.Input(ui.Type("range"), ui.Attr("hidden-ish", "")),
		),
	)
	matchChrome(t, d)
}

func lengthCSS(l style.Length) string {
	switch l.Kind {
	case style.Auto:
		return "auto"
	case style.MinContent:
		return "min-content"
	case style.MaxContent:
		return "max-content"
	case style.None:
		return "none"
	case style.Fr:
		return num(l.Px) + "fr"
	case style.FitContent:
		if l.Px == 0 && l.Percent == 0 {
			return "fit-content"
		}
		arg := style.Length{Kind: style.Fixed, Px: l.Px, Percent: l.Percent, IsPercent: l.IsPercent}
		return "fit-content(" + lengthCSS(arg) + ")"
	case style.Fixed:
		// a length or percentage, as below
	}
	switch {
	case !l.IsPercent:
		return px(l.Px)
	case l.Px == 0:
		return num(l.Percent) + "%"
	case l.Px < 0:
		return "calc(" + num(l.Percent) + "% - " + px(-l.Px) + ")"
	}
	return "calc(" + num(l.Percent) + "% + " + px(l.Px) + ")"
}

func trackCSS(t style.Track) string {
	if t.Min == t.Max {
		return lengthCSS(t.Min)
	}
	return "minmax(" + lengthCSS(t.Min) + ", " + lengthCSS(t.Max) + ")"
}

func tracksCSS(rs []style.TrackRepeat) string {
	if len(rs) == 0 {
		return "none"
	}
	var parts []string
	for _, r := range rs {
		var ts []string
		for _, t := range r.Tracks {
			ts = append(ts, trackCSS(t))
		}
		switch r.Count {
		case 1:
			parts = append(parts, ts...)
		case style.AutoFill:
			parts = append(parts, "repeat(auto-fill, "+strings.Join(ts, " ")+")")
		case style.AutoFit:
			parts = append(parts, "repeat(auto-fit, "+strings.Join(ts, " ")+")")
		default:
			parts = append(parts, "repeat("+strconv.Itoa(r.Count)+", "+strings.Join(ts, " ")+")")
		}
	}
	return strings.Join(parts, " ")
}

func autoTracksCSS(ts []style.Track) string {
	var parts []string
	for _, t := range ts {
		parts = append(parts, trackCSS(t))
	}
	if len(parts) == 0 {
		return "auto"
	}
	return strings.Join(parts, " ")
}

func lineCSS(g style.GridLine) string {
	switch {
	case g.Span > 0:
		return "span " + strconv.Itoa(g.Span)
	case g.Line != 0:
		return strconv.Itoa(g.Line)
	}
	return "auto"
}

func alignName(a style.Align) string {
	return [...]string{"normal", "auto", "stretch", "start", "end", "flex-start", "flex-end", "center", "baseline",
		"space-between", "space-around", "space-evenly"}[a]
}

func cornerCSS(c style.Size[style.Length]) string { return lengthCSS(c.Width) }

func shadowsCSS(ss []style.Shadow) string {
	if len(ss) == 0 {
		return "none"
	}
	var parts []string
	for _, s := range ss {
		p := fmt.Sprintf("%s %s %s %s %s", rgb(s.Color), px(s.X), px(s.Y), px(s.Blur), px(s.Spread))
		if s.Inset {
			p += " inset"
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, ", ")
}

func transitionsCSS(ts []style.Transition, f func(style.Transition) string) string {
	if len(ts) == 0 {
		ts = []style.Transition{{Property: "all", Easing: "ease"}}
	}
	var parts []string
	for _, t := range ts {
		parts = append(parts, f(t))
	}
	return strings.Join(parts, ", ")
}

func seconds(v float32) string { return num(v) + "s" }

// everything is every property css computes, as Chrome writes its
// computed value on an element with no box (inside display: none).
//
//nolint:gochecknoglobals // every property, a test table
var everything = func() map[string]func(s *style.Computed) string {
	m := map[string]func(s *style.Computed) string{
		"box-sizing":   func(s *style.Computed) string { return []string{"border-box", "content-box"}[s.BoxSizing] },
		"width":        func(s *style.Computed) string { return lengthCSS(s.Size.Width) },
		"height":       func(s *style.Computed) string { return lengthCSS(s.Size.Height) },
		"min-width":    func(s *style.Computed) string { return lengthCSS(s.MinSize.Width) },
		"min-height":   func(s *style.Computed) string { return lengthCSS(s.MinSize.Height) },
		"max-width":    func(s *style.Computed) string { return lengthCSS(s.MaxSize.Width) },
		"max-height":   func(s *style.Computed) string { return lengthCSS(s.MaxSize.Height) },
		"top":          func(s *style.Computed) string { return lengthCSS(s.Inset.Top) },
		"right":        func(s *style.Computed) string { return lengthCSS(s.Inset.Right) },
		"bottom":       func(s *style.Computed) string { return lengthCSS(s.Inset.Bottom) },
		"left":         func(s *style.Computed) string { return lengthCSS(s.Inset.Left) },
		"aspect-ratio": func(s *style.Computed) string { return aspect(s.AspectRatio) },
		"flex-direction": func(s *style.Computed) string {
			return []string{"row", "row-reverse", "column", "column-reverse"}[s.Direction]
		},
		"flex-wrap":             func(s *style.Computed) string { return []string{"nowrap", "wrap", "wrap-reverse"}[s.Wrap] },
		"flex-basis":            func(s *style.Computed) string { return lengthCSS(s.Basis) },
		"row-gap":               func(s *style.Computed) string { return gapCSS(s.Gap.Height) },
		"align-items":           func(s *style.Computed) string { return alignName(s.AlignItems) },
		"align-self":            func(s *style.Computed) string { return alignName(s.AlignSelf) },
		"align-content":         func(s *style.Computed) string { return alignName(s.AlignContent) },
		"justify-content":       func(s *style.Computed) string { return alignName(s.JustifyContent) },
		"justify-items":         func(s *style.Computed) string { return alignName(s.JustifyItems) },
		"justify-self":          func(s *style.Computed) string { return alignName(s.JustifySelf) },
		"grid-template-columns": func(s *style.Computed) string { return tracksCSS(s.Columns) },
		"grid-template-rows":    func(s *style.Computed) string { return tracksCSS(s.Rows) },
		"grid-auto-columns":     func(s *style.Computed) string { return autoTracksCSS(s.AutoColumns) },
		"grid-auto-rows":        func(s *style.Computed) string { return autoTracksCSS(s.AutoRows) },
		"grid-auto-flow": func(s *style.Computed) string {
			return []string{"row", "column", "row dense", "column dense"}[s.Flow]
		},
		"grid-column-start": func(s *style.Computed) string { return lineCSS(s.Column[0]) },
		"grid-column-end":   func(s *style.Computed) string { return lineCSS(s.Column[1]) },
		"grid-row-start":    func(s *style.Computed) string { return lineCSS(s.Row[0]) },
		"grid-row-end":      func(s *style.Computed) string { return lineCSS(s.Row[1]) },
		"overflow-wrap": func(s *style.Computed) string {
			return map[bool]string{true: "break-word", false: "normal"}[s.BreakWord]
		},
		"background-image":           func(s *style.Computed) string { return image(s.BackgroundImage.Source) },
		"border-right-color":         func(s *style.Computed) string { return rgb(s.BorderColor.Right) },
		"border-bottom-color":        func(s *style.Computed) string { return rgb(s.BorderColor.Bottom) },
		"border-bottom-width":        func(s *style.Computed) string { return px(s.Border.Bottom) },
		"border-left-width":          func(s *style.Computed) string { return px(s.Border.Left) },
		"border-top-style":           func(s *style.Computed) string { return borderStyle(s.BorderStyle.Top) },
		"border-right-style":         func(s *style.Computed) string { return borderStyle(s.BorderStyle.Right) },
		"border-left-style":          func(s *style.Computed) string { return borderStyle(s.BorderStyle.Left) },
		"border-top-left-radius":     func(s *style.Computed) string { return cornerCSS(s.Radius.TopLeft) },
		"border-top-right-radius":    func(s *style.Computed) string { return cornerCSS(s.Radius.TopRight) },
		"border-bottom-right-radius": func(s *style.Computed) string { return cornerCSS(s.Radius.BottomRight) },
		"border-bottom-left-radius":  func(s *style.Computed) string { return cornerCSS(s.Radius.BottomLeft) },
		"border-image-source":        func(s *style.Computed) string { return image(s.BorderImage.Source) },
		"border-image-slice": func(s *style.Computed) string {
			b := s.BorderImage.Slice
			switch {
			case b.Left != b.Right:
				return fmt.Sprintf("%s %s %s %s", num(b.Top), num(b.Right), num(b.Bottom), num(b.Left))
			case b.Bottom != b.Top:
				return fmt.Sprintf("%s %s %s", num(b.Top), num(b.Right), num(b.Bottom))
			case b.Right != b.Top:
				return fmt.Sprintf("%s %s", num(b.Top), num(b.Right))
			case b.Top == 0:
				return "100%"
			}
			return num(b.Top)
		},
		"border-image-repeat": func(s *style.Computed) string {
			words := map[style.ImageRepeat]string{style.RepeatStretch: "stretch", style.RepeatTile: "repeat",
				style.RepeatRound: "round", style.RepeatSpace: "space"}
			r := s.BorderImageRepeat
			if r[0] == r[1] {
				return words[r[0]]
			}
			return words[r[0]] + " " + words[r[1]]
		},
		"box-shadow":    func(s *style.Computed) string { return shadowsCSS(s.Shadows) },
		"outline-style": func(s *style.Computed) string { return borderStyle(s.OutlineStyle) },
		"outline-color": func(s *style.Computed) string { return rgb(s.OutlineColor) },
		"cursor": func(s *style.Computed) string {
			if s.Cursor == "" {
				return "auto"
			}
			return s.Cursor
		},
		"transition-property": func(s *style.Computed) string {
			return transitionsCSS(s.Transitions, func(t style.Transition) string { return t.Property })
		},
		"transition-duration": func(s *style.Computed) string {
			return transitionsCSS(s.Transitions, func(t style.Transition) string { return seconds(t.Duration) })
		},
		"transition-delay": func(s *style.Computed) string {
			return transitionsCSS(s.Transitions, func(t style.Transition) string { return seconds(t.Delay) })
		},
		"transition-timing-function": func(s *style.Computed) string {
			return transitionsCSS(s.Transitions, func(t style.Transition) string { return t.Easing })
		},
	}
	for k, f := range compared {
		if _, ok := m[k]; !ok {
			m[k] = f
		}
	}
	for _, side := range []string{"margin-bottom", "margin-left", "padding-right", "padding-bottom"} {
		m[side] = sideCSS(side)
	}
	for _, p := range []string{"padding-top", "padding-left", "margin-top", "margin-right"} {
		m[p] = sideCSS(p)
	}
	m["column-gap"] = func(s *style.Computed) string { return gapCSS(s.Gap.Width) }
	m["text-shadow"] = func(s *style.Computed) string {
		if len(s.TextShadows) == 0 {
			return "none"
		}
		var parts []string
		for _, sh := range s.TextShadows {
			parts = append(parts, fmt.Sprintf("%s %s %s %s", rgb(sh.Color), px(sh.X), px(sh.Y), px(sh.Blur)))
		}
		return strings.Join(parts, ", ")
	}
	m["image-rendering"] = func(s *style.Computed) string { return []string{"auto", "pixelated"}[s.ImageRendering] }
	m["accent-color"] = func(s *style.Computed) string {
		if s.AccentAuto {
			return "auto"
		}
		return rgb(s.AccentColor)
	}
	m["caret-color"] = func(s *style.Computed) string { return rgb(s.CaretColor) }
	m["object-fit"] = func(s *style.Computed) string {
		return []string{"fill", "contain", "cover", "none", "scale-down"}[s.ObjectFit]
	}
	m["object-position"] = func(s *style.Computed) string {
		p := s.ObjectPosition
		return positionCSS(p.X, p.FromRight) + " " + positionCSS(p.Y, p.FromBottom)
	}
	m["transform-origin"] = func(s *style.Computed) string {
		return lengthCSS(s.TransformOrigin.Width) + " " + lengthCSS(s.TransformOrigin.Height)
	}
	return m
}()

func sideCSS(name string) func(s *style.Computed) string {
	return func(s *style.Computed) string {
		sides := s.Margin
		if strings.HasPrefix(name, "padding") {
			sides = s.Padding
		}
		switch {
		case strings.HasSuffix(name, "top"):
			return lengthCSS(sides.Top)
		case strings.HasSuffix(name, "right"):
			return lengthCSS(sides.Right)
		case strings.HasSuffix(name, "bottom"):
			return lengthCSS(sides.Bottom)
		}
		return lengthCSS(sides.Left)
	}
}

func gapCSS(l style.Length) string {
	if l.Kind == style.Auto {
		return "normal"
	}
	return lengthCSS(l)
}

func aspect(r float32) string {
	if r == 0 {
		return "auto"
	}
	return num(r)
}

func image(src string) string {
	if src == "" {
		return "none"
	}
	if b, ok := css.DefaultImages()[src]; ok {
		return `url("data:image/png;base64,` + base64.StdEncoding.EncodeToString(b) + `")`
	}
	return `url(` + src + `)`
}

func borderStyle(b style.BorderStyle) string {
	return []string{"none", "solid", "dashed", "dotted", "inset", "outset"}[b]
}

func TestComputedValuesMatchChrome(t *testing.T) {
	fsys := fstest.MapFS{"values.css": {Data: []byte(`
:root { --w: 40%; }
.a {
  box-sizing: content-box; width: calc(100% - 20px); height: 3em; min-width: min-content; max-width: var(--w);
  min-height: 10px; max-height: fit-content; inset: 1px auto 50% calc(10% + 2px); position: relative;
  aspect-ratio: 16 / 9; margin: 1px 2% auto -3px; padding: 0.5em 1rem 3px 4%;
  display: flex; flex-flow: column-reverse wrap; flex: 2 0 30%; gap: 6px 12%;
  align-items: center; align-self: flex-end; align-content: space-between;
  justify-content: space-evenly; justify-items: start; justify-self: end; order: -2;
}
.g {
  display: grid; grid-template-columns: 10px minmax(0, 1fr) repeat(auto-fill, 20px);
  grid-template-rows: repeat(2, 1fr 2em) fit-content(40%); grid-auto-columns: 50px;
  grid-auto-rows: min-content max-content; grid-auto-flow: column dense; grid-column: 2 / span 3; grid-row: -1;
  z-index: -3; overflow: clip auto;
}
.t {
  font-family: "Go Mono", monospace; font-size: larger; font-weight: 300; font-style: italic;
  line-height: 20px; letter-spacing: -1px; color: oklch(70% 0.1 200); text-align: right;
  white-space: pre-wrap; overflow-wrap: break-word; text-overflow: ellipsis; visibility: hidden; cursor: grab;
  text-transform: capitalize;
}
.l {
  background-color: rgb(1 2 3 / 0.25); border: 2px dashed steelblue; border-left: 4px dotted;
  border-right-color: transparent; border-radius: 3px 10% 0 1em; outline: 1px solid Lime; outline-offset: -2px;
  box-shadow: 1px 2px 3px black, inset 0 0 4px 1px; opacity: 75%;
  transition: opacity 200ms, color 1s ease-in 0.5s, transform 3s cubic-bezier(0.1, 0.7, 1, 0.1);
  border-image-slice: 8 4; background-image: url(img/panel.png); border-image-source: url("img/frame 2.png");
  border-image-repeat: round repeat;
}
.x1 { transform: translate(10px, -4px) rotate(30deg) scale(2, 0.5); transform-origin: left 25%; }
.x2 { transform: skew(10deg, 0.1rad) translateY(1em) scaleX(3); transform-origin: bottom; }
.x3 { transform: matrix(1, 0.5, -0.5, 1, 7, 8) rotate(0.25turn); transform-origin: 3px 4px; }
.ts {
  text-shadow: 1px 2px 3px #000, 0 0 4px; image-rendering: pixelated; accent-color: #ffce76; caret-color: currentColor;
}
.f1 { object-fit: cover; object-position: right 4px bottom 10%; }
.f2 { object-fit: scale-down; object-position: left; }
.l2 { border-width: thin medium thick 0; border-style: solid; transition: none; flex: none; margin: 0 auto; }
.i { padding: inherit; color: initial; width: unset; border-top-color: inherit; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "values.css"))
	d.Body().Append(
		ui.Div(ui.Style("display: none"),
			ui.Div(ui.Class("a"), ui.Span(ui.Class("i"))),
			ui.Div(ui.Class("g")),
			ui.P(ui.Class("t"), ui.Span()),
			ui.Div(ui.Class("l"), ui.Span(ui.Class("i"))),
			ui.Div(ui.Class("l2")),
			ui.Button(ui.Class("t", "l")),
			ui.Div(ui.Class("x1")), ui.Div(ui.Class("x2")),
			ui.Div(ui.Class("f1")), ui.Div(ui.Class("f2")),
			ui.Div(ui.Class("ts"), ui.Span(ui.Style("color: #0f0"))),
			ui.Details(ui.Summary(ui.Text("list item"))), ui.Details(ui.Summary(ui.Style("display: block"), ui.Text("block"))),
		),
	)
	matchChromeProps(t, d, everything, 2)
}

var oklchRE = regexp.MustCompile(`oklch\(([-\d.e]+) ([-\d.e]+) ([-\d.e]+)(?: / ([-\d.e]+))?\)`)

// oklchToRGB rewrites Chrome's oklch() colours as rgb(), as ui computes
// them.
func oklchToRGB(s string) string {
	return oklchRE.ReplaceAllStringFunc(s, func(m string) string {
		p := oklchRE.FindStringSubmatch(m)
		f := func(i int) float64 { v, _ := strconv.ParseFloat(p[i], 64); return v }
		l, c, h := f(1), f(2), f(3)*math.Pi/180
		a, b := c*math.Cos(h), c*math.Sin(h)
		lr := l + 0.3963377774*a + 0.2158037573*b
		mr := l - 0.1055613458*a - 0.0638541728*b
		sr := l - 0.0894841775*a - 1.2914855480*b
		l3, m3, s3 := lr*lr*lr, mr*mr*mr, sr*sr*sr
		lin := [3]float64{
			4.0767416621*l3 - 3.3077115913*m3 + 0.2309699292*s3,
			-1.2684380046*l3 + 2.6097574011*m3 - 0.3413193965*s3,
			-0.0041960863*l3 - 0.7034186147*m3 + 1.7076147010*s3,
		}
		var ch [3]int
		for i, v := range lin {
			v = min(max(v, 0), 1)
			if v <= 0.0031308 {
				v *= 12.92
			} else {
				v = 1.055*math.Pow(v, 1/2.4) - 0.055
			}
			ch[i] = int(math.Round(v * 255))
		}
		if p[4] != "" && f(4) < 1 {
			return fmt.Sprintf("rgba(%d, %d, %d, %s)", ch[0], ch[1], ch[2], p[4])
		}
		return fmt.Sprintf("rgb(%d, %d, %d)", ch[0], ch[1], ch[2])
	})
}

// matrixCSS composes transform functions into Chrome's matrix(), in
// points (translations here have no percentages).
func matrixCSS(ops []style.TransformOp) string {
	if len(ops) == 0 {
		return "none"
	}
	m := [6]float64{1, 0, 0, 1, 0, 0}
	mul := func(n [6]float64) {
		m = [6]float64{
			m[0]*n[0] + m[2]*n[1], m[1]*n[0] + m[3]*n[1],
			m[0]*n[2] + m[2]*n[3], m[1]*n[2] + m[3]*n[3],
			m[0]*n[4] + m[2]*n[5] + m[4], m[1]*n[4] + m[3]*n[5] + m[5],
		}
	}
	for _, op := range ops {
		a := func(i int) float64 { return float64(op.A[i]) }
		switch op.Kind {
		case style.Translate:
			mul([6]float64{1, 0, 0, 1, float64(op.X.Px), float64(op.Y.Px)})
		case style.ScaleOp:
			mul([6]float64{a(0), 0, 0, a(1), 0, 0})
		case style.Rotate:
			mul([6]float64{math.Cos(a(0)), math.Sin(a(0)), -math.Sin(a(0)), math.Cos(a(0)), 0, 0})
		case style.Skew:
			mul([6]float64{1, math.Tan(a(1)), math.Tan(a(0)), 1, 0, 0})
		case style.Matrix:
			mul([6]float64{a(0), a(1), a(2), a(3), a(4), a(5)})
		}
	}
	parts := make([]string, 6)
	for i, v := range m {
		parts[i] = strconv.FormatFloat(v, 'f', 6, 64)
	}
	return "matrix(" + strings.Join(parts, ", ") + ")"
}

func TestTransformsMatchChrome(t *testing.T) {
	fsys := fstest.MapFS{"t.css": {Data: []byte(`
div { width: 100px; height: 20px; }
.x1 { transform: translate(10px, -4px) rotate(30deg) scale(2, 0.5); }
.x2 { transform: skew(10deg, 0.1rad) translateY(1em) scaleX(3); }
.x3 { transform: matrix(1, 0.5, -0.5, 1, 7, 8) rotate(0.25turn); }
.x4 { transform: scaleY(150%) skewY(-20deg) translateX(2rem); }
.x5 { transform: none; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "t.css"))
	for _, c := range []string{"x1", "x2", "x3", "x4", "x5"} {
		d.Body().Append(ui.Div(ui.Class(c)))
	}
	matchChromeProps(t, d, map[string]func(*style.Computed) string{
		"transform": func(s *style.Computed) string { return matrixCSS(s.Transform) },
	}, 2)
}

// pageURL is a url Chrome resolved against the exported page's folder.
var pageURL = regexp.MustCompile(`url\("file://[^"]*/ember-chrome-[^/"]+/([^"]*)"\)`)

var numberRE = regexp.MustCompile(`-?\d*\.?\d+(?:e-?\d+)?`)

// sameNumbers reports whether a and b differ only in how their numbers
// are written, each within 0.01.
func sameNumbers(a, b string) bool {
	if numberRE.ReplaceAllString(a, "#") != numberRE.ReplaceAllString(b, "#") {
		return false
	}
	na, nb := numberRE.FindAllString(a, -1), numberRE.FindAllString(b, -1)
	for i := range na {
		x, _ := strconv.ParseFloat(na[i], 64)
		y, _ := strconv.ParseFloat(nb[i], 64)
		if math.Abs(x-y) > 0.01 {
			return false
		}
	}
	return true
}

func TestTransitionsMatchChrome(t *testing.T) {
	fsys := fstest.MapFS{"t.css": {Data: []byte(`
div { width: 100px; height: 20px; color: #000; background-color: rgba(255, 0, 0, 0.2); opacity: 1;
  border: 2px solid navy; box-shadow: 0 0 0 #000; transform: rotate(0deg) translate(0px, 0px);
  transition: color 1s linear, background-color 1s ease-in, opacity 2s ease-out 0.1s, border-color 1s steps(4),
    box-shadow 1s cubic-bezier(0.2, 0.7, 0.4, 1), transform 1s ease-in-out; }
div.on { color: #fff; background-color: #00f; opacity: 0.2; border-color: gold;
  box-shadow: 6px 8px 10px rgba(0, 0, 0, 0.5); transform: rotate(90deg) translate(40px, 10px); }
div.on > span { color: inherit; }
div.x { transform: rotate(10deg); }
div.x.on { transform: scale(2, 1.5) skewX(20deg) translate(30px, -6px); }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "t.css"))
	d.Body().Append(ui.Div(ui.Span(ui.Text("label"))), ui.Div(ui.Class("x")))
	d.Layout(geom.V2(800, 600))
	tr := css.NewTransitions()
	tr.Compute(d.HTML(), d.Sheets())
	shownProps := map[string]func(*style.Computed) string{
		"color":             compared["color"],
		"background-color":  compared["background-color"],
		"opacity":           compared["opacity"],
		"border-top-color":  compared["border-top-color"],
		"border-left-color": compared["border-left-color"],
		"box-shadow":        everything["box-shadow"],
		"transform":         func(s *style.Computed) string { return matrixCSS(s.Transform) },
	}
	script := `els.forEach(e => getComputedStyle(e).color);
	document.querySelectorAll("div").forEach(e => e.classList.add("on"));
	els.forEach(e => getComputedStyle(e).color);
	document.getAnimations().forEach(a => { a.pause(); a.currentTime = 350; });`
	then := func() {
		d.HTML().Walk(func(n *dom.Node) bool {
			if n.Tag() == dom.Div {
				n.AddClass("on")
			}
			return true
		})
		tr.Compute(d.HTML(), d.Sheets())
		tr.Step(0.35)
	}
	matchChromeShown(t, d, shownProps, 2, script, then, func(n *dom.Node) *style.Computed { return dom.Data(n).Shown })
}

// positionCSS is a background-position axis as Chrome computes it: an offset from the end
// edge is calc(100% - offset).
func positionCSS(l style.Length, fromEnd bool) string {
	if !fromEnd {
		return lengthCSS(l)
	}
	if l.IsPercent && l.Px == 0 {
		return num(100-l.Percent) + "%"
	}
	return "calc(100% - " + lengthCSS(l) + ")"
}

func TestBackgroundsAndPlaceholdersMatchChrome(t *testing.T) {
	fsys := fstest.MapFS{"b.css": {Data: []byte(`
.a { background: url(img/panel.png) no-repeat right 6px center; }
.b { background-position: bottom 4px left; }
.c { background-position: 10px 20%; background-repeat: no-repeat; }
.form { color: #123456; }
.form input::placeholder { color: currentColor; }
textarea::placeholder { color: rebeccapurple; }
.form::selection { background-color: rgba(255, 207, 118, 0.4); }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "b.css"))
	d.Body().Append(
		ui.Div(ui.Class("a")), ui.Div(ui.Class("b")), ui.Div(ui.Class("c")),
		ui.Select(ui.Option(ui.Text("one"))),
		ui.Div(ui.Class("form"), ui.Input(ui.Attr("placeholder", "who"))),
		ui.Textarea(ui.Attr("placeholder", "notes")),
		ui.Input(ui.Attr("placeholder", "plain")),
	)
	raw, err := chrome.Eval(d, geom.V2(800, 600), `return Array.from(document.body.querySelectorAll("*")).map(e => {
		const s = getComputedStyle(e), p = getComputedStyle(e, "::placeholder");
		const sel = getComputedStyle(e, "::selection");
		return [s.backgroundRepeat, s.backgroundPositionX, s.backgroundPositionY, s.backgroundImage, p.color,
			e.classList.contains("form") ? sel.backgroundColor : ""];
	});`)
	if err != nil {
		t.Fatal(err)
	}
	var want [][6]string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	d.Layout(geom.V2(800, 600))
	var got []*dom.Node
	d.Body().Walk(func(n *dom.Node) bool {
		if n != d.Body() && !n.IsText() {
			got = append(got, n)
		}
		return true
	})
	if len(got) != len(want) {
		t.Fatalf("ember has %d elements, Chrome %d", len(got), len(want))
	}
	for i, n := range got {
		s := dom.Data(n).Style
		repeat := "repeat"
		if s.BackgroundNoRepeat {
			repeat = "no-repeat"
		}
		selection := ""
		if s.HasSelection {
			selection = rgb(s.SelectionColor)
		}
		ours := [6]string{repeat,
			positionCSS(s.BackgroundPosition.X, s.BackgroundPosition.FromRight),
			positionCSS(s.BackgroundPosition.Y, s.BackgroundPosition.FromBottom),
			image(s.BackgroundImage.Source), rgb(s.PlaceholderColor), selection}
		w := want[i]
		if !s.HasPlaceholder || n.Tag() != dom.Input && n.Tag() != dom.Textarea {
			ours[4], w[4] = "", ""
		}
		for k := range w {
			if !same("", ours[k], w[k]) {
				t.Errorf("%s %d: ember %q, Chrome %q", where(n), k, ours[k], w[k])
			}
		}
	}
}

// TestTheUASheetMatchesChrome compares the elements whose look comes from
// the user-agent sheet with Chrome's own: hr, p, the headings, monospace
// text, disabled controls and a non-modal dialog, as computed (in a
// display: none box, as the computed-values test does).
func TestTheUASheetMatchesChrome(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Div(ui.Style("display: none"),
		ui.Hr(),
		ui.P(ui.Text("p")), ui.H1(ui.Text("h1 "), ui.Code(ui.Text("code"))), ui.H2(ui.Text("h2")), ui.H3(ui.Text("h3")),
		ui.H4(ui.Text("h4")),
		ui.Code(ui.Text("code")), ui.Kbd(ui.Text("kbd")), ui.Samp(ui.Text("samp")), ui.Pre(ui.Text("pre")),
		ui.Button(ui.Disabled(), ui.Text("b")), ui.Input(ui.Type("text"), ui.Disabled(), ui.Value("x")),
		ui.Select(ui.Disabled(), ui.Option(ui.Disabled(), ui.Text("o"))), ui.Textarea(ui.Disabled(), ui.Text("t")),
		ui.Dialog(ui.Open(), ui.Text("dialog")),
	))
	matchChromeProps(t, d, uaProps, 3)
}

// uaProps are what the user-agent sheet sets for those elements.
//
//nolint:gochecknoglobals // the user-agent sheet's properties, a test table
var uaProps = func() map[string]func(*style.Computed) string {
	m := map[string]func(*style.Computed) string{}
	for _, p := range []string{"display", "position", "opacity", "white-space", "color", "background-color",
		"font-size", "font-weight", "font-family", "border-image-slice"} {
		m[p] = everything[p]
	}
	for _, side := range []string{"top", "right", "bottom", "left"} {
		for _, p := range []string{"margin-", "padding-", "border-%s-width", "border-%s-style", "border-%s-color"} {
			name := p + side
			if strings.Contains(p, "%s") {
				name = fmt.Sprintf(p, side)
			}
			m[name] = everything[name]
		}
	}
	return m
}()

// TestAModalDialogMatchesChromes compares a dialog shown modal with
// Chrome's: fixed, held inside the viewport, and its ::backdrop.
func TestAModalDialogMatchesChromes(t *testing.T) {
	d := ui.NewDocument()
	modal := ui.Dialog(ui.Text("modal"))
	d.Body().Append(modal)
	d.Layout(geom.V2(800, 600))
	d.ShowModal(modal)
	props := map[string]func(*style.Computed) string{
		"position":         compared["position"],
		"border-top-width": compared["border-top-width"],
		"background-color": compared["background-color"],
		"padding-top":      everything["padding-top"],
		"overflow-y":       everything["overflow-y"],
	}
	matchChromeProps(t, d, props, 2)
	if got := dom.Data(modal).Style.Backdrop; rgb(got) != "rgba(0, 0, 0, 0.1)" {
		t.Errorf("backdrop %s, Chrome rgba(0, 0, 0, 0.1)", rgb(got))
	}
}

func TestAFocusedElementRestylesItsDescendantsAsChrome(t *testing.T) {
	fsys := fstest.MapFS{"t.css": {Data: []byte(`
.child, .grand, .deep { display: none; }
.lb:focus > .child { display: block; }
.lb:focus > .b > .grand { display: block; }
.lb:focus .deep { display: block; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "t.css"))
	lb := ui.Div(ui.Class("lb"), ui.Attr("tabindex", "0"),
		ui.Span(ui.Class("child"), ui.Text("child")),
		ui.Div(ui.Class("b"), ui.Span(ui.Class("grand"), ui.Text("grand")),
			ui.Div(ui.Span(ui.Class("deep"), ui.Text("deep")))))
	d.Body().Append(lb)
	d.Layout(geom.V2(800, 600))
	then := func() {
		d.HTML().Walk(func(n *dom.Node) bool {
			if n.HasClass("lb") {
				n.SetState(dom.Focus, true)
			}
			return true
		})
		d.Layout(geom.V2(800, 600))
	}
	props := map[string]func(*style.Computed) string{"display": compared["display"]}
	matchChromeShown(t, d, props, 2, `document.querySelector(".lb").focus();`, then,
		func(n *dom.Node) *style.Computed { return dom.Data(n).Style })
}

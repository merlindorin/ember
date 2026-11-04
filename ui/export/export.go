// Package export writes a Document as an HTML page with its stylesheets
// and fonts, which Chrome lays out as ui does: the reference of the
// layout tests, and pages for the docs.
package export

import (
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/css"
	"github.com/merlindorin/ember/ui/internal/assets"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/fonts"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/merlindorin/ember/ui/layout"
)

// Option adds to the exported page.
type Option func(*page)

type page struct {
	head     []string
	snapshot func(n *ui.Node) ([]byte, bool)
}

// Head adds raw markup to the page's head, such as a test's script.
func Head(markup string) Option { return func(p *page) { p.head = append(p.head, markup) } }

// Snapshots gives each canvas the PNG of what ember drew in its content
// box, at the device scale, for Chrome to draw in it.
func Snapshots(of func(n *ui.Node) ([]byte, bool)) Option {
	return func(p *page) { p.snapshot = of }
}

// Write writes d as a self-contained HTML page: the user-agent sheet,
// the fonts as data URLs, the document's sheets, then its tree.
func Write(w io.Writer, d *ui.Document, opts ...Option) error {
	var p page
	for _, o := range opts {
		o(&p)
	}
	assets.Styled(d)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html")
	attrs(&b, d, d.HTML())
	b.WriteString(">\n<head>\n<meta charset=\"utf-8\">\n<style>\n")
	for _, f := range fonts.Of(d).Faces() {
		style := "normal"
		if f.Italic {
			style = "italic"
		}
		fmt.Fprintf(&b, "@font-face { font-family: %q; font-weight: %d; font-style: %s; "+
			"src: url(data:font/ttf;base64,%s); }\n",
			f.Family, f.Weight, style, base64.StdEncoding.EncodeToString(f.TTF))
	}
	set := fonts.Of(d)
	for _, f := range set.Faces() {
		if r := set.FallbackRanges(f); r != "" {
			style := "normal"
			if f.Italic {
				style = "italic"
			}
			fmt.Fprintf(&b, "@font-face { font-family: %q; font-weight: %d; font-style: %s; "+
				"src: url(data:font/ttf;base64,%s); unicode-range: %s; }\n",
				f.Family, f.Weight, style, base64.StdEncoding.EncodeToString(set.Fallback().TTF), r)
		}
	}
	b.WriteString(classicScrollbars)
	b.WriteString(bareBars)
	b.WriteString("@layer ember-user-agent {\n" + states(inline(d, css.Default, "")) + "}\n")
	b.WriteString(truncateStart)
	for _, s := range d.Sheets() {
		b.WriteString(states(inline(d, s.String(), path.Dir(s.Name))))
	}
	b.WriteString("</style>\n")
	b.WriteString(showModals)
	b.WriteString(scrollTo)
	b.WriteString(drawSnapshots)
	for _, h := range p.head {
		b.WriteString(h)
		b.WriteByte('\n')
	}
	b.WriteString("</head>\n")
	node(&b, &p, d, d.Body())
	b.WriteString("\n</html>\n")
	_, err := io.WriteString(w, b.String())
	return err
}

// tr is d's translator as layout takes it, nil for none.
func tr(d *ui.Document) func(string) string {
	if t := d.Translator(); t != nil {
		return t.Tr
	}
	return nil
}

func node(b *strings.Builder, p *page, d *ui.Document, n *dom.Node) {
	if n.IsText() {
		b.WriteString(html.EscapeString(layout.Translated(n, tr(d))))
		return
	}
	b.WriteByte('<')
	b.WriteString(n.Tag().String())
	attrs(b, d, n)
	if n.Tag() == dom.Canvas && p.snapshot != nil {
		if data, ok := p.snapshot(n); ok {
			fmt.Fprintf(b, ` data-ember-snapshot="data:image/png;base64,%s"`, base64.StdEncoding.EncodeToString(data))
		}
	}
	b.WriteByte('>')
	if n.Tag().Void() {
		return
	}
	if ph, ok := n.Attr("placeholder"); ok && n.Tag() == dom.Select && d.Selected(n) < 0 {
		fmt.Fprintf(b, `<option value="" disabled selected hidden data-ember-implied="">%s</option>`,
			html.EscapeString(layout.TranslatedAt(n, ph, tr(d))))
	}
	cut := slices.Contains(n.Classes(), "truncate-start")
	if cut {
		b.WriteString(`<bdi dir="ltr" data-ember-implied="">`)
	}
	for _, c := range n.Children() {
		node(b, p, d, c)
	}
	if cut {
		b.WriteString("</bdi>")
	}
	b.WriteString("</")
	b.WriteString(n.Tag().String())
	b.WriteByte('>')
}

// attrs writes n's attributes, a control's as it is now: an input's
// value and checked, a select's chosen option. A select's placeholder,
// a ui extension, becomes the hidden first option Chrome shows instead.
func attrs(b *strings.Builder, d *ui.Document, n *dom.Node) {
	write := func(name, value string) {
		fmt.Fprintf(b, " %s=\"%s\"", name, html.EscapeString(value))
	}
	for _, name := range []string{"id", "class"} {
		if v, ok := n.Attr(name); ok {
			write(name, v)
		}
	}
	for _, p := range pseudo {
		if n.State().Has(p.state) {
			write("data-ember-"+p.name, "")
		}
	}
	if s := dom.Data(n).Scroll; s.X != 0 || s.Y != 0 {
		write("data-ember-scroll", fmt.Sprintf("%g %g", s.X, s.Y))
	}
	style, _ := n.Attr("style")
	if thumb := switchThumb(d, n); thumb != "" {
		style = strings.TrimPrefix(strings.TrimSpace(style+"; "+thumb), "; ")
	}
	if bar := barLook(d, n); bar != "" {
		style = strings.TrimPrefix(strings.TrimSpace(style+"; "+bar), "; ")
	}
	if c, ok := placeholderColor(d, n); ok {
		style = strings.TrimSpace(style + "; color: " + c)
		style = strings.TrimPrefix(style, "; ")
	}
	if style != "" {
		write("style", inline(d, style, "."))
	}
	typ, _ := n.Attr("type")
	var layer uint64
	modal := false
	if n.Tag() == dom.Dialog {
		layer, modal = assets.Modal(d, n)
	}
	if modal {
		write("data-ember-modal", strconv.FormatUint(layer, 10))
	}
	live := map[string]bool{
		"open": modal, "style": true, "checked": n.Tag() == dom.Input, "value": n.Tag() == dom.Input && typ != "checkbox",
		"placeholder": n.Tag() == dom.Select, "selected": n.Tag() == dom.Option,
	}
	for _, a := range n.Attrs() {
		if live[a.Name] {
			continue
		}
		if n.Tag() == dom.Img && a.Name == "src" {
			if data, ok := assets.Of(d, a.Value); ok {
				write("src", "data:"+http.DetectContentType(data)+";base64,"+base64.StdEncoding.EncodeToString(data))
				continue
			}
		}
		if a.Name == "placeholder" {
			write(a.Name, layout.TranslatedAt(n, a.Value, tr(d)))
			continue
		}
		write(a.Name, a.Value)
	}
	switch {
	case n.Tag() == dom.Input && (typ == "checkbox" || typ == "radio"):
		if d.Checked(n) {
			write("checked", "")
		}
	case n.Tag() == dom.Input:
		if v := d.Value(n); v != "" {
			write("value", v)
		}
	case n.Tag() == dom.Option:
		if p := n.Parent(); p != nil && p.Tag() == dom.Select && optionIndex(p, n) == d.Selected(p) {
			write("selected", "")
		}
	}
}

func optionIndex(sel, opt *dom.Node) int {
	i := 0
	for _, c := range sel.Children() {
		if c == opt {
			return i
		}
		if c.Tag() == dom.Option {
			i++
		}
	}
	return -1
}

var urls = regexp.MustCompile(`url\(\s*(['"]?)([^'")]+)['"]?\s*\)`)

// inline replaces the urls of css, relative to dir as css resolves a
// sheet's (dir "" for none, "." for a style attribute), by data URLs of
// the document's images, so the page carries them.
func inline(d *ui.Document, src, dir string) string {
	return urls.ReplaceAllStringFunc(src, func(m string) string {
		url := urls.FindStringSubmatch(m)[2]
		name := url
		if !strings.Contains(url, ":") && dir != "" && dir != "." {
			name = path.Join(dir, url)
		}
		b, ok := assets.Of(d, name)
		if !ok {
			return m
		}
		return fmt.Sprintf("url(data:%s;base64,%s)", http.DetectContentType(b), base64.StdEncoding.EncodeToString(b))
	})
}

// switchThumb is the declarations that draw a switch's thumb in Chrome,
// which draws none for the switch attribute: a circle inset 2 points in
// the track, in the element's colour, at its start or, checked, its end,
// as paint draws it.
func switchThumb(d *ui.Document, n *dom.Node) string {
	if n.Tag() != dom.Input {
		return ""
	}
	if _, ok := n.Attr("switch"); !ok {
		return ""
	}
	st := dom.Data(n).Style
	if st == nil {
		return ""
	}
	h, ok := st.Size.Height.Resolve(0)
	if !ok || st.Size.Height.Percent != 0 {
		return ""
	}
	pad := func(l style.Length) float32 { v, _ := l.Resolve(0); return v }
	if st.BoxSizing == style.BorderBox {
		h -= st.Border.Top + st.Border.Bottom + pad(st.Padding.Top) + pad(st.Padding.Bottom)
	}
	r := max(h/2-2, 1)
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g">`+
		`<circle cx="%g" cy="%g" r="%g" fill="%s" fill-opacity="%g"/></svg>`,
		h, h, h/2, h/2, r, st.Color.Hex(), st.Color.A)
	side := "left"
	if d.Checked(n) {
		side = "right"
	}
	return fmt.Sprintf("background-image: url(data:image/svg+xml;base64,%s); background-repeat: no-repeat; "+
		"background-origin: content-box; background-position: %s center",
		base64.StdEncoding.EncodeToString([]byte(svg)), side)
}

// bareBars takes Chrome's own look off ranges and progress bars, which
// barLook draws as paint does. Its selectors weigh nothing (:where), so
// an author's rules on the same elements win as they do in ember.
const bareBars = `:where(input[type="range"]) { appearance: none; background: none; }
:where(input[type="range"])::-webkit-slider-runnable-track { background: none; }
:where(input[type="range"])::-webkit-slider-thumb { appearance: none; width: 0; height: 0; }
:where(progress) { appearance: none; background: none; }
:where(progress)::-webkit-progress-bar, :where(progress)::-webkit-progress-value { background: none; }
`

// barLook is the declarations that draw a range or a progress bar in
// Chrome as ember's convention has paint draw them: its background-color
// the track, its accent-color the fill to the value, and a range's thumb,
// 14 across on the value, in its color, the accent while hovered. A
// range's is an SVG border image outset each side by the thumb's
// overhang, in place of any border; a progress bar's an SVG background
// in its content box, inside its border. Its size is the laid-out Rect,
// so a document exported before any layout shows Chrome's bare bars.
func barLook(d *ui.Document, n *dom.Node) string {
	st := dom.Data(n).Style
	if st == nil {
		return ""
	}
	typ, _ := n.Attr("type")
	isRange := n.Tag() == dom.Input && strings.EqualFold(typ, "range")
	if !isRange && n.Tag() != dom.Progress {
		return ""
	}
	r := n.Box()
	pad := func(l style.Length) float32 { v, _ := l.Resolve(r.W); return v }
	x0 := st.Border.Left + pad(st.Padding.Left)
	y0 := st.Border.Top + pad(st.Padding.Top)
	w := r.W - x0 - st.Border.Right - pad(st.Padding.Right)
	h := r.H - y0 - st.Border.Bottom - pad(st.Padding.Bottom)
	if w <= 0 || h <= 0 {
		return ""
	}
	fill := func(c color.Color) string {
		return fmt.Sprintf(`fill="%s" fill-opacity="%g"`, c.Hex(), c.A)
	}
	track := st.Background
	if track.A <= 0 {
		track = st.Color
		track.A *= 0.25
	}
	accent := st.AccentColor
	if st.AccentAuto {
		accent = st.Color
	}
	var svg strings.Builder
	var out float32
	if isRange {
		f := rangeFraction(d, n)
		th, thumb := min(6, h), min(14, h)
		out = thumb / 2
		ink := st.Color
		if n.State().Has(dom.Hover) {
			ink = accent
		}
		fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g">`, w+2*out, h)
		fmt.Fprintf(&svg, `<rect x="%g" y="%g" width="%g" height="%g" rx="%g" %s/>`, out, (h-th)/2, w, th, th/2, fill(track))
		if f > 0 {
			fmt.Fprintf(&svg, `<rect x="%g" y="%g" width="%g" height="%g" rx="%g" %s/>`,
				out, (h-th)/2, w*f, th, th/2, fill(accent))
		}
		fmt.Fprintf(&svg, `<circle cx="%g" cy="%g" r="%g" %s/>`, out+w*f, h/2, out, fill(ink))
	} else {
		fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g">`, w, h)
		fmt.Fprintf(&svg, `<rect x="0" y="0" width="%g" height="%g" %s/>`, w, h, fill(track))
		if f, ok := progressFraction(n); ok && f > 0 {
			fmt.Fprintf(&svg, `<rect x="0" y="0" width="%g" height="%g" %s/>`, w*f, h, fill(accent))
		}
	}
	svg.WriteString("</svg>")
	url := "url(data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg.String())) + ")"
	if !isRange {
		return "background: " + url + " no-repeat content-box"
	}
	// A range keeps its background-color, as the cascade computed it,
	// but clipped to its text, of which it has none, so only the
	// border-image draws.
	return fmt.Sprintf("-webkit-background-clip: text; background-clip: text; border-image-source: %s; "+
		"border-image-slice: 0 fill; border-image-width: 0; border-image-outset: 0 %gpx", url, out)
}

// rangeFraction is where a range's value lies between its min and max,
// 0 to 100 by default, as the Document has it now.
func rangeFraction(d *ui.Document, n *dom.Node) float32 {
	lo, hi := attrFloat(n, "min", 0), attrFloat(n, "max", 100)
	v, err := strconv.ParseFloat(d.Value(n), 64)
	if err != nil || hi <= lo {
		return 0
	}
	return float32(min(max((v-lo)/(hi-lo), 0), 1))
}

// progressFraction is a progress bar's value over its max (1 by default),
// not ok when it has no value: indeterminate.
func progressFraction(n *dom.Node) (float32, bool) {
	s, ok := n.Attr("value")
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	hi := attrFloat(n, "max", 1)
	if err != nil || hi <= 0 {
		return 0, false
	}
	return float32(min(max(v/hi, 0), 1)), true
}

func attrFloat(n *dom.Node, name string, def float64) float64 {
	if s, ok := n.Attr(name); ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return def
}

// placeholderColor is the ::placeholder colour of a select showing its
// placeholder, which Chrome doesn't give a select.
func placeholderColor(d *ui.Document, n *dom.Node) (string, bool) {
	if n.Tag() != dom.Select || d.Selected(n) >= 0 {
		return "", false
	}
	if _, ok := n.Attr("placeholder"); !ok {
		return "", false
	}
	st := dom.Data(n).Style
	if st == nil || !st.HasPlaceholder {
		return "", false
	}
	c := st.PlaceholderColor
	if c.A >= 1 {
		return c.Hex(), true
	}
	s := c.SRGB()
	return fmt.Sprintf("rgba(%d, %d, %d, %.3g)", int(s[0]*255+0.5), int(s[1]*255+0.5), int(s[2]*255+0.5), c.A), true
}

// pseudo are the pseudo-classes Chrome can't be put in by a page, which
// export writes as data-ember-<name> attributes and selectors also match.
var pseudo = []struct { //nolint:gochecknoglobals // a read-only table
	name  string
	state dom.State
}{{"hover", dom.Hover}, {"active", dom.Active}, {"focus-visible", dom.FocusVisible}, {"focus", dom.Focus}}

var pseudoClasses = regexp.MustCompile(`:(hover|active|focus-visible|focus)\b`)

// states makes css's :hover, :active, :focus and :focus-visible also match
// the elements export marks as in that state, so Chrome draws a pointer's
// or the focus's look in a still page.
func states(css string) string {
	return pseudoClasses.ReplaceAllString(css, ":is(:$1, [data-ember-$1])")
}

// showModals opens with showModal the dialogs the Document has open modal,
// in the order of their layers, which stacks them in the top layer as
// paint does, and gives them their ::backdrop.
const (
	showModals = `<script>addEventListener("DOMContentLoaded", () => ` +
		`[...document.querySelectorAll("dialog[data-ember-modal]")]` +
		`.sort((a, b) => a.dataset.emberModal - b.dataset.emberModal).forEach(d => d.showModal()));</script>` + "\n"
)

// scrollTo scrolls each scroller to the offset the Document has it at,
// once the page is parsed and again once it has loaded, its fonts with it.
const scrollTo = `<script>const emberScroll = () => document.querySelectorAll("[data-ember-scroll]").forEach(e => { ` +
	`const [x, y] = e.dataset.emberScroll.split(" ").map(Number); e.scrollLeft = x; e.scrollTop = y; });
addEventListener("DOMContentLoaded", emberScroll); ` +
	`addEventListener("load", () => document.fonts.ready.then(emberScroll));</script>` + "\n"

// drawSnapshots draws each canvas's snapshot into it at the device scale:
// its used size held as its CSS size, so the bitmap set to its content box
// in device pixels leaves the layout alone.
const drawSnapshots = `<script>addEventListener("load", () => ` +
	`document.querySelectorAll("canvas[data-ember-snapshot]").forEach(c => {
const s = getComputedStyle(c);
c.style.width = s.width; c.style.height = s.height;
const w = c.clientWidth - parseFloat(s.paddingLeft) - parseFloat(s.paddingRight), ` +
	`h = c.clientHeight - parseFloat(s.paddingTop) - parseFloat(s.paddingBottom);
c.width = Math.round(w * devicePixelRatio); c.height = Math.round(h * devicePixelRatio);
const img = new Image(); img.onload = () => c.getContext("2d").drawImage(img, 0, 0, c.width, c.height); ` +
	`img.src = c.dataset.emberSnapshot;
}));</script>` + "\n"

// truncateStart has Chrome's ellipsis cut the start of a truncate-start
// element, as layout does: right to left outside, its content isolated
// left to right inside (a bdi marked data-ember-implied, which the tree
// doesn't have) so a path's slashes stay where they are. It comes before
// the author's sheets, which may override it.
const truncateStart = ".truncate-start { direction: rtl; text-align: left; }\n"

// classicScrollbars has Chrome reserve the 15-point gutter css gives
// scrolling boxes (style.Box.ScrollbarWidth), not overlay scrollbars; the
// page itself shows none, as a window's document doesn't.
const classicScrollbars = "::-webkit-scrollbar { width: 15px; height: 15px; }\n" +
	"html::-webkit-scrollbar { display: none; }\n" + thumbs

// thumbs draws Chrome's scrollbar thumbs as paint draws ember's: 6 points
// wide, 2 from the far edge and 4 from the track's ends, rounded by 3, in
// white at a quarter; the transparent borders place it in the gutter.
const thumbs = "::-webkit-scrollbar-thumb { background-color: rgba(255, 255, 255, 0.25); " +
	"background-clip: padding-box; border: solid transparent; }\n" +
	"::-webkit-scrollbar-thumb:vertical { border-width: 4px 2px 4px 7px; border-radius: 10px 5px 5px 10px / 7px; }\n" +
	"::-webkit-scrollbar-thumb:horizontal { border-width: 7px 4px 2px 4px; border-radius: 7px / 10px 10px 5px 5px; }\n" +
	"::-webkit-scrollbar-corner { background: transparent; }\n"

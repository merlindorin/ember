package paint

import (
	"strconv"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/textedit"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// The marks a control shows of its state, which CSS can't draw with
// appearance: none: a checkbox's tick, a switch's thumb, a range's track
// and thumb, a bar's fill, a summary's triangle in its
// marker box, a text field's caret and selection, and a modal dialog's
// backdrop. They draw in the element's colour, over its background,
// inside its content box.

//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var (
	selectionColor = color.Hex("#3390ff66")
	// meterColors are a meter's fill by region, Chrome's green, amber and
	// red.
	meterColors = map[control.Region]color.Color{
		control.Optimum:      color.Hex("#107c10"),
		control.Suboptimal:   color.Hex("#ffb900"),
		control.EvenLessGood: color.Hex("#d83b01"),
	}
)

// progressPeriod is how long an indeterminate progress's segment takes
// across its bar, in seconds.
const progressPeriod = 1.5

// faint is how much of the colour the empty part of a track or bar takes.
const faint = 0.25

func (p *painter) machine(n *dom.Node) any {
	if p.control == nil || n == nil {
		return nil
	}
	return p.control(n)
}

func inputType(n *dom.Node) string {
	if t, ok := n.Attr("type"); ok {
		return t
	}
	return "text"
}

// marked reports whether b is a control that draws marks.
func marked(b *box.Box) bool {
	if b.Kind != box.Element || b.Node == nil {
		return false
	}
	t := b.Node.Tag()
	return t == dom.Input || t == dom.Textarea || t == dom.Progress || t == dom.Meter ||
		t == dom.Select || t == dom.Img || t == dom.Canvas
}

// modal reports whether b is a dialog shown modal, with a backdrop.
func (p *painter) modal(b *box.Box) bool {
	if b.Kind != box.Element || b.Node == nil || b.Node.Tag() != dom.Dialog {
		return false
	}
	d, ok := p.machine(b.Node).(*control.Dialog)
	return ok && d.Open && d.Modal
}

func (p *painter) layer(b *box.Box) uint64 {
	d, _ := p.machine(b.Node).(*control.Dialog)
	return d.Layer
}

func content(e entry) geom.Rect {
	c := e.b.ContentRect()
	c.X, c.Y = c.X+e.at.X, c.Y+e.at.Y
	return c
}

func (p *painter) marks(e entry, alpha float32) {
	n, st := e.b.Node, shown(e.b)
	if !st.Visible {
		return
	}
	if p.culled(e, inkOf(e, st)) {
		if n.Tag() == dom.Select {
			p.open(e)
		}
		return
	}
	ink := fade(st.Color, alpha)
	c := content(e)
	switch n.Tag() {
	case dom.Input:
		switch inputType(n) {
		case "checkbox":
			p.checkbox(n, c, ink)
		case "radio":
			p.radio(n, c, ink)
		case "range":
			p.rangeMarks(n, c, st, alpha)
		default:
			p.caret(e, caretInk(st, alpha))
		}
	case dom.Textarea:
		p.caret(e, caretInk(st, alpha))
	case dom.Progress:
		f, known := progressOf(p.machine(n), n)
		p.bar(c, f, trackInk(st, alpha), accentInk(st, alpha))
		if !known && p.timed && p.record(livePart{kind: liveSlide, e: e, rect: c, ink: ink}, c) {
			p.slide(c, ink)
		}
	case dom.Meter:
		m := meterOf(p.machine(n), n)
		p.bar(c, float32(m.Fraction()), trackInk(st, alpha), fade(meterColors[m.Region()], alpha))
	case dom.Img:
		p.img(e, alpha)
	case dom.Canvas:
		p.canvasElement(e, c)
	case dom.Select:
		p.open(e)
	case dom.TextTag, dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
		dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label, dom.Button,
		dom.Option, dom.Details, dom.Summary, dom.Dialog, dom.Table, dom.Thead, dom.Tbody, dom.Tr, dom.Th, dom.Td, dom.Hr,
		dom.Kbd, dom.Samp, dom.Pre, dom.Caption, dom.Tfoot: // only marked controls come here
	}
}

// radioGap is the space Chrome leaves between a checked radio's ring and
// its dot.
const radioGap = 1.5

// radio draws a checked radio's dot in its colour, filling the content box
// but radioGap.
func (p *painter) radio(n *dom.Node, c geom.Rect, ink color.Color) {
	checked := n.State().Has(dom.Checked)
	if m, ok := p.machine(n).(*control.Radio); ok {
		checked = m.Checked
	}
	if r := min(c.W, c.H)/2 - radioGap; checked && r > 0 {
		p.s.Circle(c.Center(), r, ink)
	}
}

func (p *painter) checkbox(n *dom.Node, c geom.Rect, ink color.Color) {
	checked, mixed := n.State().Has(dom.Checked), false
	if m, ok := p.machine(n).(*control.Checkbox); ok {
		checked, mixed = m.Checked, m.Indeterminate
	}
	if _, ok := n.Attr("switch"); ok {
		p.switchThumb(c, checked, ink)
		return
	}
	at := func(x, y float32) geom.Vec2 { return geom.V2(c.X+x*c.W, c.Y+y*c.H) }
	switch {
	case mixed:
		h := max(2, c.H*0.12)
		p.s.Rect(geom.Rect{X: c.X + 0.2*c.W, Y: c.Y + (c.H-h)/2, W: 0.6 * c.W, H: h}, ink)
	case checked:
		w := max(1.5, c.W/8)
		p.s.Line(at(0.2, 0.5), at(0.42, 0.72), w, ink)
		p.s.Line(at(0.42, 0.72), at(0.8, 0.3), w, ink)
	}
}

// switchThumb draws a switch's thumb, inset 2 points in its track, at
// the start while off and the end while on. The track is the element's
// background, from CSS.
func (p *painter) switchThumb(c geom.Rect, on bool, ink color.Color) {
	r := c.H / 2
	x := c.X + r
	if on {
		x = c.X + c.W - r
	}
	p.s.Circle(geom.V2(x, c.Y+r), max(r-2, 1), ink)
}

// rangeMarks draws a range as ember's convention has it: a track 6
// high (or the box's height) along the middle in its background-color,
// filled to the value in its accent-color, and a thumb 14 across centred
// on the value, overhanging the ends as Chrome's does, in its color, in
// its accent-color while hovered.
func (p *painter) rangeMarks(n *dom.Node, c geom.Rect, st *style.Computed, alpha float32) {
	r, ok := p.machine(n).(*control.Range)
	if !ok {
		r = rangeOf(n)
	}
	f := float32(r.Fraction())
	h := min(rangeTrack, c.H)
	cy := c.Y + c.H/2
	track := geom.Rect{X: c.X, Y: cy - h/2, W: c.W, H: h}
	p.s.RoundRect(track, h/2, trackInk(st, alpha))
	accent := accentInk(st, alpha)
	if f > 0 {
		fill := track
		fill.W *= f
		p.s.RoundRect(fill, h/2, accent)
	}
	thumb := fade(st.Color, alpha)
	if n.State().Has(dom.Hover) {
		thumb = accent
	}
	p.s.Circle(geom.V2(c.X+f*c.W, cy), min(rangeThumb, c.H)/2, thumb)
}

// rangeTrack and rangeThumb are a range's track height and thumb size.
const (
	rangeTrack = 6
	rangeThumb = 14
)

// bar draws a progress or meter bar: its track, then f of it filled.
func (p *painter) bar(c geom.Rect, f float32, track, fill color.Color) {
	p.s.Rect(c, track)
	if f > 0 {
		v := c
		v.W *= min(f, 1)
		p.s.Rect(v, fill)
	}
}

// tracked reports whether b is a range, a progress or a meter, whose
// background-color is its track, painted with its marks, not behind its
// box.
func tracked(b *box.Box) bool {
	if b.Node == nil {
		return false
	}
	switch b.Node.Tag() {
	case dom.Progress, dom.Meter:
		return true
	case dom.Input:
		return inputType(b.Node) == "range"
	// Elements that track nothing.
	case dom.TextTag, dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
		dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label, dom.Button,
		dom.Select, dom.Option, dom.Textarea, dom.Details, dom.Summary, dom.Dialog, dom.Table, dom.Thead, dom.Tbody, dom.Tr,
		dom.Th, dom.Td, dom.Img, dom.Hr, dom.Canvas, dom.Kbd, dom.Samp, dom.Pre, dom.Caption, dom.Tfoot:
	}
	return false
}

// trackInk is a track's colour: the background-color, else the ink
// faint.
func trackInk(st *style.Computed, alpha float32) color.Color {
	if st.Background.A > 0 {
		return fade(st.Background, alpha)
	}
	return fade(fade(st.Color, faint), alpha)
}

// accentInk is accent-color, the element's colour when auto.
func accentInk(st *style.Computed, alpha float32) color.Color {
	if st.AccentAuto {
		return fade(st.Color, alpha)
	}
	return fade(st.AccentColor, alpha)
}

// marker draws a summary's disclosure triangle as Chrome paints its
// ::marker: Blink's canonical triangle in a square 0.657em wide whose
// top is 0.670em above the baseline, from the marker's start, pointing
// right while closed and down while open, in the summary's colour.
func (p *painter) marker(e entry, alpha float32) {
	st := shown(e.b)
	disclosure := st.ListStyle == style.ListDisclosureClosed || st.ListStyle == style.ListDisclosureOpen
	if !st.Visible || len(e.b.Lines) == 0 || !disclosure || p.culled(e, inkOf(e, st)) {
		return
	}
	ink := fade(st.Color, alpha)
	if ink.A <= 0 {
		return
	}
	size := markerSize * st.Font.Size
	at := geom.V2(e.at.X, e.at.Y+e.b.Lines[0].Baseline-markerRise*st.Font.Size)
	shape := closedMarker
	if st.ListStyle == style.ListDisclosureOpen {
		shape = openMarker
	}
	pts := make([]geom.Vec2, len(shape))
	for i, q := range shape {
		pts[i] = at.Add(q.Scale(size))
	}
	p.s.Triangles(pts, []int{0, 1, 2}, ink)
}

// markerSize and markerRise are the marker's square and how far its top
// is above the baseline, in ems, as Chrome draws it.
const (
	markerSize = 0.657
	markerRise = 0.670
)

//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var (
	closedMarker = []geom.Vec2{{X: 0, Y: 0}, {X: 0.86, Y: 0.5}, {X: 0, Y: 1}}
	openMarker   = []geom.Vec2{{X: 0, Y: 0.07}, {X: 0.5, Y: 0.93}, {X: 1, Y: 0.07}}
)

// editing is a focused text field's machine, laid out, nil otherwise.
func (p *painter) editing(b *box.Box) *control.Text {
	if !marked(b) || !b.Node.State().Has(dom.Focus) {
		return nil
	}
	var t *control.Text
	switch m := p.machine(b.Node).(type) {
	case *control.Text:
		t = m
	case *control.Number:
		t = &m.Text
	}
	if t == nil || t.Layout == nil {
		return nil
	}
	return t
}

// rowed is a text layout that knows each line's rectangle in its text
// run's space, a textarea's, whose empty lines have no fragment.
type rowed interface{ Row(line int) geom.Rect }

// lineBox is where line i of b's text lies, in the root's space: the
// layout's row or its text run's fragment, else the content box.
func lineBox(e entry, l textedit.Layout, i int) geom.Rect {
	b := e.b
	origin := e.at.Sub(b.Scroll)
	for _, c := range b.Children {
		if c.Kind != box.TextRun {
			continue
		}
		if r, ok := l.(rowed); ok {
			f := r.Row(i)
			return geom.Rect{X: origin.X + c.Pos.X + f.X, Y: origin.Y + c.Pos.Y + f.Y, W: f.W, H: f.H}
		}
		if i < len(c.Lines) {
			f := c.Lines[i].Rect
			return geom.Rect{X: origin.X + c.Pos.X + f.X, Y: origin.Y + c.Pos.Y + f.Y, W: f.W, H: f.H}
		}
	}
	r := content(e)
	r.X -= b.Scroll.X
	return r
}

// selection draws a focused text field's selection, behind its text, in
// its ::selection background-color, else the default.
func (p *painter) selection(e entry, alpha float32) {
	t := p.editing(e.b)
	if t == nil {
		return
	}
	from, to := t.Editor.Selection()
	if from == to {
		return
	}
	c := selectionColor
	if st := shown(e.b); st.HasSelection {
		c = st.SelectionColor
	}
	l := t.Layout
	p.s.PushClip(padding(e.b, e.at))
	for line := l.Line(from); line <= l.Line(to); line++ {
		start, end := l.Span(line)
		a, z := max(from, start), min(to, end)
		lb := lineBox(e, l, line)
		x0, x1 := lb.X+l.X(a), lb.X+l.X(z)
		if x1 > x0 {
			p.s.Rect(geom.Rect{X: x0, Y: lb.Y, W: x1 - x0, H: lb.H}, fade(c, alpha))
		}
	}
	p.s.PopClip()
}

// caretInk is caret-color, the text's colour when auto.
func caretInk(st *style.Computed, alpha float32) color.Color {
	if st.CaretAuto {
		return fade(st.Color, alpha)
	}
	return fade(st.CaretColor, alpha)
}

// caret draws a focused, editable text field's caret, a live part.
func (p *painter) caret(e entry, ink color.Color) {
	t := p.editing(e.b)
	if t == nil || t.Editor.ReadOnly {
		return
	}
	if p.record(livePart{kind: liveCaret, e: e, ink: ink}, intersect(caretRect(e, t).Inset(-1), padding(e.b, e.at))) {
		p.drawCaret(e, ink)
	}
}

// caretRect is the caret's bar: a point wide, its line high, where the
// editor's caret is.
func caretRect(e entry, t *control.Text) geom.Rect {
	i := t.Editor.Caret()
	lb := lineBox(e, t.Layout, t.Layout.Line(i))
	return geom.Rect{X: lb.X + t.Layout.X(i), Y: lb.Y, W: 1, H: lb.H}
}

// drawCaret draws the caret in its blink's lit half-second.
func (p *painter) drawCaret(e entry, ink color.Color) {
	t := p.editing(e.b)
	if t == nil || int(p.time*2)%2 != 0 {
		return
	}
	p.s.PushClip(padding(e.b, e.at))
	p.s.Rect(caretRect(e, t), ink)
	p.s.PopClip()
}

func attr(n *dom.Node, name string) (float64, bool) {
	s, ok := n.Attr(name)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

// rangeOf is a range as its attributes set it, at the middle of its
// range without a value, as HTML does.
func rangeOf(n *dom.Node) *control.Range {
	r := control.NewRange()
	if v, ok := attr(n, "min"); ok {
		r.Min = v
	}
	if v, ok := attr(n, "max"); ok {
		r.Max = v
	}
	if v, ok := attr(n, "step"); ok {
		r.Step = v
	}
	v, ok := attr(n, "value")
	if !ok {
		v = r.Min + (r.Max-r.Min)/2
	}
	r.SetValue(v)
	return r
}

// progressOf is how full a progress is, and false when it is
// indeterminate.
func progressOf(m any, n *dom.Node) (float32, bool) {
	if pr, ok := m.(*control.Progress); ok {
		return float32(pr.Fraction()), !pr.Indeterminate
	}
	v, ok := attr(n, "value")
	if !ok {
		return 0, false
	}
	pr := &control.Progress{Value: v}
	pr.Max, _ = attr(n, "max")
	return float32(pr.Fraction()), true
}

// slide draws an indeterminate progress's segment, a third of its bar,
// crossing it every progressPeriod.
func (p *painter) slide(c geom.Rect, ink color.Color) {
	seg := c.W / 3
	phase := p.time/progressPeriod - float32(int(p.time/progressPeriod))
	p.s.PushClip(c)
	p.s.Rect(geom.Rect{X: c.X + phase*(c.W+seg) - seg, Y: c.Y, W: seg, H: c.H}, ink)
	p.s.PopClip()
}

func meterOf(m any, n *dom.Node) *control.Meter {
	if mt, ok := m.(*control.Meter); ok {
		return mt
	}
	v, _ := attr(n, "value")
	mt := control.NewMeter(v)
	if x, ok := attr(n, "min"); ok {
		mt.Min = x
	}
	if x, ok := attr(n, "max"); ok {
		mt.Max = x
	}
	mt.Low, mt.High, mt.Optimum = mt.Min, mt.Max, mt.Min+(mt.Max-mt.Min)/2
	if x, ok := attr(n, "low"); ok {
		mt.Low = x
	}
	if x, ok := attr(n, "high"); ok {
		mt.High = x
	}
	if x, ok := attr(n, "optimum"); ok {
		mt.Optimum = x
	}
	return mt
}

// canvasElement has a canvas element's machine draw in its content box c,
// on a twod.Canvas: c's top-left its origin, c's size its Size, clipped to
// it. Its last drawing is replayed while its version, place, transform
// and scale hold. Live, it is a part drawn apart over a replay. A hit
// test's walk draws none.
func (p *painter) canvasElement(e entry, c geom.Rect) {
	m, ok := p.machine(e.b.Node).(*control.Canvas)
	if !ok || m.Draw == nil || p.probing || c.W <= 0 || c.H <= 0 {
		return
	}
	if p.record(livePart{kind: liveCanvas, e: e, rect: c}, c) {
		p.drawCanvas(m, c)
	}
}

func (p *painter) drawCanvas(m *control.Canvas, c geom.Rect) {
	tc, onto := p.s.(*twod.Canvas)
	if !onto {
		return
	}
	xf, size := tc.Transform(), tc.Size()
	tc.PushClip(c)
	tc.SetTransform(xf.Mul(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(c.X, c.Y)}))
	k := &m.Kept
	moved := k.Version != m.Version || k.At != c || k.Transform != xf || k.Scale != p.scale
	if !k.Ok || moved || !tc.Replay(&k.Recording) {
		tc.SetSize(geom.V2(c.W, c.H))
		mark := tc.Mark()
		m.Draw(tc)
		k.Ok = tc.Keep(mark, &k.Recording)
		k.Version, k.At, k.Transform, k.Scale = m.Version, c, xf, p.scale
		tc.SetSize(size)
	}
	tc.SetTransform(xf)
	tc.PopClip()
}

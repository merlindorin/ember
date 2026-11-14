package ui

import (
	"slices"
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// line is a one-line text field's text laid out: where each position is
// along it. It measures the text as drawn and answers in the value's
// positions, through cases when text-transform changed the case.
type line struct {
	xs []float32
	// src, font and case are what xs was measured from, so a frame where
	// none changed keeps the line.
	src   []rune
	font  style.Font
	tt    style.TextTransform
	cases *caseMap
}

func newLine(text []rune, f style.Font, fonts box.Fonts) *line {
	l := &line{xs: make([]float32, len(text)+1), font: f}
	for i := range text {
		l.xs[i+1] = fonts.Advance(f, string(text[:i+1]))
	}
	return l
}

func (l *line) Lines() int          { return 1 }
func (l *line) Line(int) int        { return 0 }
func (l *line) Span(int) (int, int) { return 0, l.cases.valueAt(len(l.xs) - 1) }
func (l *line) X(i int) float32     { return l.xs[min(max(l.cases.drawnAt(i), 0), len(l.xs)-1)] }
func (l *line) Page() int           { return 1 }
func (l *line) At(_ int, x float32) int {
	best := 0
	for i, at := range l.xs {
		if abs(at-x) < abs(l.xs[best]-x) {
			best = i
		}
	}
	return l.cases.valueAt(best)
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// layText gives each text field's machine the layout of what it shows,
// for its caret, its selection and the clicks placing them.
func (d *Document) layText() {
	for n, m := range d.machines {
		t := textOf(m)
		if t == nil {
			continue
		}
		s := dom.Data(n).Style
		if s == nil {
			continue
		}
		src := t.Editor.Runes()
		if t.Multiline() {
			d.layArea(n, t, s.Font)
			continue
		}
		if l, ok := t.Layout.(*line); ok && slices.Equal(l.src, src) && sameFont(l.font, s.Font) && l.tt == s.TextTransform {
			continue
		}
		shown, cases := d.shownText(n, t)
		if !t.Empty() {
			n.SetShadow(shown, false)
		}
		l := newLine([]rune(shown), s.Font, d.fonts)
		l.src, l.tt, l.cases = slices.Clone(src), s.TextTransform, cases
		t.Layout = l
	}
}

// layArea cuts a textarea's text along the lines layout gave it, once
// its text, font, width or case changed; lines laid out from other text
// than it draws wait for the next layout.
func (d *Document) layArea(n *Node, t *control.Text, f style.Font) {
	if d.tree == nil {
		return
	}
	b := d.tree.Of[n]
	if b == nil {
		return
	}
	src := t.Editor.Runes()
	width := b.ContentRect().W
	tt := dom.Data(n).Style.TextTransform
	if a, ok := t.Layout.(*area); ok && a.width == width && slices.Equal(a.src, src) && sameFont(a.font, f) && a.tt == tt {
		return
	}
	var run *box.Box
	for _, c := range b.Children {
		if c.Kind == box.TextRun {
			run = c
			break
		}
	}
	m := d.fonts.Metrics(f)
	rows := int(b.ContentRect().H / max(m.Ascent+m.Descent+m.LineGap, 1))
	if run != nil && len(run.Lines) > 0 {
		rows = int(b.ContentRect().H / max(run.Lines[0].Rect.H, 1))
	}
	shown, cases := d.shownText(n, t)
	if sh := dom.Data(n).Shadow; !t.Empty() && sh != nil && sh.Text() != shown {
		n.SetShadow(shown, false)
		return
	}
	a := newArea([]rune(shown), run, f, d.fonts, rows)
	a.src, a.width, a.tt, a.cases = slices.Clone(src), width, tt, cases
	t.Layout = a
}

// shownText is a text field's text as it draws: bullets for a password, else
// its value as its text-transform cases it, with the map between them.
func (d *Document) shownText(n *Node, t *control.Text) (string, *caseMap) {
	st := dom.Data(n).Style
	if t.Password() || st == nil {
		return t.Shown(), nil
	}
	return fieldCase(t.Shown(), st.TextTransform)
}

func sameFont(a, b style.Font) bool {
	return a.Size == b.Size && a.Weight == b.Weight && a.Italic == b.Italic && slices.Equal(a.Family, b.Family)
}

// caretAt places a text field's caret where at falls, in the document's
// space: on a textarea's line under it; extend keeps the selection's
// other end. A drag past the field's edge reaches the lines beyond, which
// followCaret then scrolls in.
func (d *Document) caretAt(n *Node, t *control.Text, at geom.Vec2, extend bool) {
	b := d.tree.Of[n]
	if b == nil || t.Layout == nil {
		return
	}
	c := b.ContentRect()
	x := at.X - n.Box().X - c.X + b.Scroll.X
	line := 0
	if a, ok := t.Layout.(*area); ok {
		var run geom.Vec2
		for _, k := range b.Children {
			if k.Kind == box.TextRun {
				run = k.Pos
				break
			}
		}
		y := at.Y - n.Box().Y + b.Scroll.Y - run.Y
		for line < a.Lines()-1 && y >= a.Row(line).Y+a.Row(line).H {
			line++
		}
		x = at.X - n.Box().X + b.Scroll.X - run.X - a.Row(line).X
	}
	t.Editor.Place(t.Layout.At(line, x), extend)
}

// area is a textarea's text as layout cut it into lines: each line's
// first and last position, where each position is along it, and the
// line's rectangle in its text run's space. A line without text, which
// layout gives no fragment, stands a line's height under the one before.
type area struct {
	starts, ends []int
	xs           [][]float32
	rows         []geom.Rect
	page         int
	// src, font, width and case are what it was cut from; its positions
	// are the drawn text's, cases mapping the value's to them.
	src   []rune
	font  style.Font
	width float32
	tt    style.TextTransform
	cases *caseMap
}

func (a *area) Lines() int     { return len(a.starts) }
func (a *area) Line(i int) int { return a.line(a.cases.drawnAt(i)) }
func (a *area) line(j int) int {
	k, _ := slices.BinarySearch(a.starts, j+1)
	return max(k-1, 0)
}
func (a *area) Span(line int) (int, int) {
	return a.cases.valueAt(a.starts[line]), a.cases.valueAt(a.ends[line])
}
func (a *area) X(i int) float32 {
	j := a.cases.drawnAt(i)
	k := a.line(j)
	xs := a.xs[k]
	return xs[min(max(j-a.starts[k], 0), len(xs)-1)]
}
func (a *area) Page() int { return a.page }
func (a *area) At(line int, x float32) int {
	line = min(max(line, 0), len(a.starts)-1)
	best := 0
	for i, at := range a.xs[line] {
		if abs(at-x) < abs(a.xs[line][best]-x) {
			best = i
		}
	}
	return a.cases.valueAt(a.starts[line] + best)
}

// Row is line's rectangle in its text run's space.
func (a *area) Row(line int) geom.Rect { return a.rows[min(max(line, 0), len(a.rows)-1)] }

// newArea cuts text along run's lines, the fragments layout made of it:
// a hard break starts a line, a soft one drops the spaces it broke at.
func newArea(text []rune, run *box.Box, f style.Font, fonts box.Fonts, rowsShown int) *area {
	a := &area{font: f}
	m := fonts.Metrics(f)
	lh := m.Ascent + m.Descent + m.LineGap
	var frags []box.Fragment
	if run != nil {
		frags = run.Lines
		if len(frags) > 0 {
			lh = frags[0].Rect.H
		}
	}
	next := float32(0)
	add := func(start int, shown []rune, r geom.Rect) {
		xs := make([]float32, len(shown)+1)
		for j := range shown {
			xs[j+1] = fonts.Advance(f, string(shown[:j+1]))
		}
		a.starts, a.ends = append(a.starts, start), append(a.ends, start+len(shown))
		a.xs, a.rows = append(a.xs, xs), append(a.rows, r)
		next = r.Y + r.H
	}
	fi, p := 0, 0
	for {
		end := p
		for end < len(text) && text[end] != '\n' {
			end++
		}
		q, first := p, true
		for q < end && fi < len(frags) {
			fr := frags[fi]
			shown := []rune(strings.TrimSuffix(fr.Text, emptyLine))
			for !first && q < end && !hasPrefix(text[q:end], shown) && text[q] == ' ' {
				q++
			}
			if !hasPrefix(text[q:end], shown) {
				break
			}
			add(q, shown, fr.Rect)
			q, first = q+len(shown), false
			fi++
		}
		if first {
			if fi < len(frags) && strings.TrimSuffix(frags[fi].Text, emptyLine) == "" && frags[fi].Rect.Y <= next+0.5 {
				add(p, nil, frags[fi].Rect)
				fi++
			} else {
				add(p, nil, geom.Rect{Y: next, H: lh})
			}
		} else if q < end {
			last := len(a.starts) - 1
			a.ends[last] = end
			for _, r := range text[q:end] {
				a.xs[last] = append(a.xs[last], a.xs[last][len(a.xs[last])-1]+fonts.Advance(f, string(r)))
			}
		}
		if end >= len(text) {
			break
		}
		p = end + 1
	}
	a.page = max(rowsShown, 1)
	return a
}

// emptyLine is what layout holds a textarea's line after its last break
// with: no text of its own.
const emptyLine = "\u00a0"

func hasPrefix(s, prefix []rune) bool {
	return len(s) >= len(prefix) && slices.Equal(s[:len(prefix)], prefix)
}

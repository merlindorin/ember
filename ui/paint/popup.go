package paint

import (
	"sync"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// An open select's list draws over everything, after the whole tree, like
// a modal's top layer: under the select, above it when it doesn't fit,
// kept inside the root, in the select's look: its background, border and
// colour, the highlighted option in its accent-color with its text in the
// select's colour, or Chrome's blue and white while the accent is auto.

//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var (
	highlight     = color.Hex("#0075ff")
	highlightText = color.White
	popupShadow   = color.RGBA(0, 0, 0, 0.25)
)

const (
	// optionPad is the room either side of an option's label, and dimmed
	// what a disabled option's colour keeps.
	optionPad = 6
	dimmed    = 0.4
)

// popup is an open select met while painting.
type popup struct {
	b  *box.Box
	at geom.Vec2
	m  *control.Select
	xf geom.Transform2D
}

// menu is where an open select's options lie, in the root's space.
type menu struct {
	rect        geom.Rect
	border      float32
	row         float32
	first, rows int
}

// rowHeight is an option row's height: the font's line, and 2 points
// above and below.
func rowHeight(fonts box.Fonts, f style.Font) float32 {
	line := f.Size * 1.25
	if fonts != nil {
		m := fonts.Metrics(f)
		line = m.Ascent + m.Descent + m.LineGap
	}
	return line + 4
}

func (p *painter) open(e entry) {
	if m, ok := p.machine(e.b.Node).(*control.Select); ok && m.Open {
		p.popups = append(p.popups, popup{b: e.b, at: e.at, m: m, xf: p.s.Transform()})
	}
}

// place lays an open select's list out within the viewport.
func (p *painter) place(pp popup) menu {
	b, st := pp.b, shown(pp.b)
	n := len(pp.m.Options)
	l := menu{border: b.Border.Top, row: rowHeight(p.fonts, st.Font), rows: n}
	w := b.Size.X
	if p.fonts != nil {
		for _, o := range pp.m.Options {
			w = max(w, p.fonts.Advance(st.Font, o.Text())+2*optionPad+2*l.border)
		}
	}
	vp := p.viewport
	h := float32(n)*l.row + 2*l.border
	if h > vp.H {
		l.rows = max(int((vp.H-2*l.border)/l.row), 1)
		h = float32(l.rows)*l.row + 2*l.border
	}
	y := pp.at.Y + b.Size.Y
	switch {
	case y+h <= vp.Y+vp.H:
	case pp.at.Y-h >= vp.Y:
		y = pp.at.Y - h
	default:
		y = vp.Y + vp.H - h
	}
	x := min(max(pp.at.X, vp.X), vp.X+vp.W-w)
	l.rect = geom.Rect{X: x, Y: y, W: w, H: h}
	l.first = max(0, min(pp.m.Highlight-l.rows+1, n-l.rows))
	return l
}

// rowRect is option i's row, if shown.
func (l menu) rowRect(i int) (geom.Rect, bool) {
	if i < l.first || i >= l.first+l.rows {
		return geom.Rect{}, false
	}
	return geom.Rect{
		X: l.rect.X + l.border, Y: l.rect.Y + l.border + float32(i-l.first)*l.row,
		W: l.rect.W - 2*l.border, H: l.row,
	}, true
}

func (l menu) inner() geom.Rect { return l.rect.Inset(l.border) }

// popups draws the open selects' lists, on top of everything.
func (p *painter) drawPopups() {
	saved := p.s.Transform()
	for _, pp := range p.popups {
		p.s.SetTransform(pp.xf)
		if p.record(livePart{kind: liveList, list: pp}, geom.Rect{}) {
			p.drawList(pp, p.place(pp))
		}
	}
	p.s.SetTransform(saved)
}

func (p *painter) drawList(pp popup, l menu) {
	st := shown(pp.b)
	radius := radiiOf(st, geom.V2(l.rect.W, l.rect.H))
	p.s.Glow(l.rect, max(radius[0], radius[1], radius[2], radius[3]), 8, popupShadow)
	bg := st.Background
	if bg.A <= 0 {
		bg = color.White
	}
	p.fill(l.rect, radius, bg)
	p.border(l.rect, pp.b.Border, radius, st, 1)
	face := twod.Regular()
	if p.face != nil {
		if f := p.face(st.Font); f != nil {
			face = f
		}
	}
	p.s.PushClip(l.inner())
	for i, o := range pp.m.Options {
		row, ok := l.rowRect(i)
		if !ok {
			continue
		}
		ink := st.Color
		if i == pp.m.Highlight {
			mark, text := highlight, highlightText
			if !st.AccentAuto {
				mark, text = st.AccentColor, st.Color
			}
			p.s.Rect(row, mark)
			ink = text
		}
		if o.Disabled {
			ink = fade(ink, dimmed)
		}
		p.s.Text(geom.V2(row.X+optionPad, row.Y+2), o.Text(),
			twod.Typeface(face), twod.FontSize(st.Font.Size), twod.Color(ink), twod.Tracking(st.LetterSpacing))
	}
	p.s.PopClip()
}

// Pick is what is under the pointer while a select is open: the select,
// the option row under it (-1 for none), and whether it is over the list
// or over the select itself. Neither is a click outside, which closes it.
type Pick struct {
	Select     *dom.Node
	Index      int
	List, Self bool
}

// PickOption is the open select's Pick at p, in the root's space; a Pick
// without a Select when none is open.
func PickOption(t *box.Tree, p geom.Vec2, opts ...Option) Pick {
	pt := scan(t, opts, p, false)
	if pt == nil {
		return Pick{Index: -1}
	}
	defer scanners.Put(pt)
	if len(pt.popups) == 0 {
		return Pick{Index: -1}
	}
	pp := pt.popups[len(pt.popups)-1]
	l := pt.place(pp)
	pick := Pick{Select: pp.b.Node, Index: -1}
	pick.Self = geom.Rect{X: pp.at.X, Y: pp.at.Y, W: pp.b.Size.X, H: pp.b.Size.Y}.Contains(p)
	if l.rect.Contains(p) {
		pick.List, pick.Self = true, false
		for i := range pp.m.Options {
			if r, ok := l.rowRect(i); ok && r.Contains(p) {
				pick.Index = i
			}
		}
	}
	return pick
}

// scan walks t in paint order without drawing, keeping the open selects
// it meets and, hitting, the last element box holding at (found). Only
// boxes whose ink holds at are walked to, and not hitting, none: the walk
// then only finds the open selects. The caller puts the painter back in
// scanners.
func scan(t *box.Tree, opts []Option, at geom.Vec2, hitting bool) *painter {
	if t == nil || t.Root == nil {
		return nil
	}
	pt := pooled[painter](&scanners)
	s := pt.probe
	if s == nil {
		s = &probe{}
	}
	*s = probe{xf: geom.Identity2D(), clips: s.clips[:0]}
	*pt = painter{s: s, probe: s, probing: true, point: at, pointless: !hitting,
		scratch: pt.scratch[:0], chain: pt.chain[:0], popups: pt.popups[:0], top: pt.top[:0], slabs: pt.slabs}
	for _, o := range opts {
		o(pt)
	}
	root := t.Root
	pt.start(root)
	pt.page(root)
	pt.use(nil)
	return pt
}

// scanners keeps scan's painters, each with its probe, between walks.
//
//nolint:gochecknoglobals // a sync.Pool every paint shares
var scanners = sync.Pool{New: func() any { return new(painter) }}

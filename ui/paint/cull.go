package paint

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// culled reports whether ink, in the root's space under the surface's
// transform now, misses what shows of e: the viewport and every clip of
// e's chain met in that same space. Under another transform only the
// clips met in it count, so nothing is culled by where it was before a
// transform moved it. A scan culls what its point misses, everything
// when it has no point. Ink not culled is checked against the live parts
// drawn before it (cover).
func (p *painter) culled(e entry, ink geom.Rect) bool {
	if p.missed(e, ink) {
		return true
	}
	p.cover(e.b, ink, false)
	return false
}

// missed is culled's test alone, for ink that must not count as covering
// a live part.
func (p *painter) missed(e entry, ink geom.Rect) bool {
	if p.probing {
		return p.pointless || !ink.Contains(p.s.Transform().Inverse().Xform(p.point))
	}
	if p.probing {
		return false
	}
	xf := p.s.Transform()
	view, bounded := geom.Rect{}, false
	if xf == p.base {
		view, bounded = p.viewport, true
	}
	for c := e.clip; c != nil; c = c.parent {
		if c.xf != xf {
			continue
		}
		if !bounded {
			view, bounded = c.rect, true
			continue
		}
		view = intersect(view, c.rect)
	}
	return bounded && !overlaps(ink, view)
}

func intersect(a, b geom.Rect) geom.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: max(x1-x0, 0), H: max(y1-y0, 0)}
}

func overlaps(a, b geom.Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// inkOf is what an element's decoration may cover: its border box grown
// by its outer shadows' reach.
func inkOf(e entry, st *style.Computed) geom.Rect {
	r := geom.Rect{X: e.at.X, Y: e.at.Y, W: e.b.Size.X, H: e.b.Size.Y}
	for _, f := range e.b.Fragments {
		r = union(r, geom.Rect{X: e.at.X + f.X, Y: e.at.Y + f.Y, W: f.W, H: f.H})
	}
	return grow(r, shadowReach(st.Shadows))
}

// textInk is what a text run's lines may cover, glyphs overhanging their
// line boxes by up to an em, its shadows included.
func textInk(e entry, st *style.Computed) geom.Rect {
	var r geom.Rect
	for i, f := range e.b.Lines {
		l := geom.Rect{X: e.at.X + f.Rect.X, Y: e.at.Y + f.Rect.Y, W: f.Rect.W, H: f.Rect.H}
		if i == 0 {
			r = l
		} else {
			r = union(r, l)
		}
	}
	return grow(r, st.Font.Size+shadowReach(st.TextShadows))
}

// shadowReach is how far outer shadows reach past their box.
func shadowReach(shadows []style.Shadow) float32 {
	var d float32
	for _, sh := range shadows {
		if !sh.Inset {
			d = max(d, max(abs(sh.X), abs(sh.Y))+sh.Spread+1.5*sh.Blur)
		}
	}
	return d
}

func union(a, b geom.Rect) geom.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

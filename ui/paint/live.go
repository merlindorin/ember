package paint

import (
	"sync"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/box"
)

// LiveSet is what a paint drew live (decision 0052): a focused editable
// field's caret, an indeterminate progress's slide, an open select's list,
// each with its place, clips and transform, so PaintLive can draw them
// again over a replay of the rest; and whether anything painted after one
// covered it, when they can't be drawn on top.
type LiveSet struct {
	parts   []livePart
	covered bool
	// rects are where the parts lie in the root's space, for the check.
	rects []geom.Rect
	// owners are the boxes the parts belong to, whose own outlines ring
	// them without covering them.
	owners []*box.Box
	// viewport is the paint's, which places an open list.
	viewport geom.Rect
	// slabs hold the parts' clip chains, copied from the paint's, which
	// the next paint reuses.
	slabs [][]clip
	used  int
}

type liveKind uint8

const (
	liveCaret liveKind = iota
	liveSlide
	liveList
	liveCanvas
)

type livePart struct {
	kind liveKind
	e    entry
	xf   geom.Transform2D
	ink  color.Color
	rect geom.Rect
	list popup
}

// Any reports whether the paint drew anything live.
func (l *LiveSet) Any() bool { return len(l.parts) > 0 }

// Timed reports whether a part moves with the clock: all but canvases,
// which change only when invalidated.
func (l *LiveSet) Timed() bool {
	for _, p := range l.parts {
		if p.kind != liveCanvas {
			return true
		}
	}
	return false
}

// Covered reports whether something painted after a live part overlaps
// it, so the parts can't go on top of a replay.
func (l *LiveSet) Covered() bool { return l.covered }

func (l *LiveSet) reset() {
	clear(l.parts)
	l.parts, l.rects, l.owners, l.covered, l.used = l.parts[:0], l.rects[:0], l.owners[:0], false, 0
}

// Live has the paint keep what it draws live in set, set emptied first.
// Apart, it draws the tree without them, for PaintLive to draw on top.
func Live(set *LiveSet, apart bool) Option {
	return func(p *painter) {
		set.reset()
		p.live, p.apart = set, apart
	}
}

// record keeps part, reporting whether it is to be drawn now: not when
// the paint keeps live parts apart. A part under another transform than
// the root's counts as covered: it can't be checked.
func (p *painter) record(part livePart, area geom.Rect) bool {
	if p.live == nil {
		return true
	}
	part.xf = p.s.Transform()
	part.e.clip = p.live.keep(part.e.clip)
	p.live.parts = append(p.live.parts, part)
	p.live.viewport = p.viewport
	if part.xf != p.base {
		p.live.covered = true
	} else if part.kind != liveList {
		p.live.rects = append(p.live.rects, area)
		p.live.owners = append(p.live.owners, part.e.b)
	}
	return !p.apart
}

// keep is a copy of the clip chain c in l's slabs.
func (l *LiveSet) keep(c *clip) *clip {
	if c == nil {
		return nil
	}
	parent := l.keep(c.parent)
	i, j := l.used/slab, l.used%slab
	if i == len(l.slabs) {
		l.slabs = append(l.slabs, make([]clip, slab))
	}
	l.used++
	l.slabs[i][j] = clip{rect: c.rect, round: c.round, radii: c.radii, xf: c.xf, parent: parent}
	return &l.slabs[i][j]
}

// cover notes ink drawn now for b: it covers a live part it overlaps,
// but an element's own outline, a ring round its box, never covers the
// parts inside the box.
func (p *painter) cover(b *box.Box, ink geom.Rect, ring bool) {
	if p.live == nil || p.live.covered {
		return
	}
	for i, r := range p.live.rects {
		if ring && p.live.owners[i] == b {
			continue
		}
		if overlaps(ink, r) {
			p.live.covered = true
			return
		}
	}
}

// painters keeps painters between paints, with their scratch room.
//
//nolint:gochecknoglobals // a sync.Pool every paint shares
var painters = sync.Pool{New: func() any { return new(painter) }}

// PaintLive draws set's live parts on s at the clock's time seconds, each
// under its own clips and transform, as Paint with opts drew them: over a
// replay of the rest of the frame.
//
//nolint:revive // Live is already the option that keeps the parts; PaintLive pairs with Paint.
func PaintLive(s Surface, set *LiveSet, time float32, opts ...Option) {
	p := pooled[painter](&painters)
	defer painters.Put(p)
	*p = painter{s: s, base: s.Transform(), scratch: p.scratch[:0], chain: p.chain[:0]}
	for _, o := range opts {
		o(p)
	}
	p.time, p.timed = time, true
	p.viewport = set.viewport
	saved := s.Transform()
	for _, part := range set.parts {
		s.SetTransform(part.xf)
		switch part.kind {
		case liveCaret:
			p.use(part.e.clip)
			p.drawCaret(part.e, part.ink)
		case liveSlide:
			p.use(part.e.clip)
			p.slide(part.rect, part.ink)
		case liveList:
			p.use(nil)
			p.drawList(part.list, p.place(part.list))
		case liveCanvas:
			p.use(part.e.clip)
			if m, ok := p.machine(part.e.b.Node).(*control.Canvas); ok {
				p.drawCanvas(m, part.rect)
			}
		}
	}
	p.use(nil)
	s.SetTransform(saved)
}

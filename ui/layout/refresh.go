package layout

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Refresh brings a laid-out tree up to date with what can change without a layout: the
// kept scroll offsets, clamped and copied into Box.Scroll, and the sticky boxes they move.
// It is for a frame in which no node is dirty and the viewport is the same, visits only
// the tree's Live boxes, and reports whether any of them moved.
func Refresh(t *box.Tree, _ *dom.Node) bool {
	if t == nil || t.Root == nil {
		return false
	}
	moved := false
	for i, b := range t.Live {
		refreshBox(b)
		if now := [2]geom.Vec2{b.Scroll, b.Sticky}; now != t.Applied[i] {
			t.Applied[i] = now
			moved = true
		}
	}
	return moved
}

// collectLive refreshes every box below b, as a fresh layout leaves them, and gathers
// the live ones and their offsets into t.
func collectLive(b *box.Box, t *box.Tree) {
	if b.Node != nil && b.Kind == box.Element || isSticky(b) {
		refreshBox(b)
	}
	if b.Node != nil && b.Kind == box.Element && b.Style != nil && isScrollContainer(b.Style) || isSticky(b) {
		t.Live = append(t.Live, b)
		t.Applied = append(t.Applied, [2]geom.Vec2{b.Scroll, b.Sticky})
	}
	for _, c := range b.Children {
		collectLive(c, t)
	}
}

func isSticky(b *box.Box) bool { return b.Style != nil && b.Style.Position == style.Sticky }

// refreshBox brings b's scroll offset and sticky offset up to date.
func refreshBox(b *box.Box) {
	if b.Node != nil && b.Kind == box.Element {
		b.Scroll = clampScroll(b)
		dom.Data(b.Node).Scroll = b.Scroll
	}
	if isSticky(b) {
		s := stickyOffset(b)
		b.Pos = b.Pos.Sub(b.Sticky).Add(s)
		b.Sticky = s
	}
}

// scrollport is a box's padding box less its scrollbar gutter, in its own space.
func scrollport(b *box.Box) geom.Rect {
	g := b.Scrollbar
	return geom.Rect{
		X: b.Border.Left, Y: b.Border.Top,
		W: b.Size.X - b.Border.Horizontal() - g.X,
		H: b.Size.Y - b.Border.Vertical() - g.Y,
	}
}

// clampScroll is a scroll container's kept offset within its scrollable overflow beyond
// the scrollport; anything else does not scroll.
func clampScroll(b *box.Box) geom.Vec2 {
	if b.Style == nil || !isScrollContainer(b.Style) || b.Style.Display == style.DisplayNone {
		return geom.Vec2{}
	}
	port := scrollport(b)
	s := dom.Data(b.Node).Scroll
	return geom.Vec2{
		X: min(max(s.X, 0), max(b.Content.X-port.W, 0)),
		Y: min(max(s.Y, 0), max(b.Content.Y-port.H, 0)),
	}
}

// flowAt is a box's border box top-left in the root's space as the flow placed it, before
// scrolling and sticky offsets.
func flowAt(b *box.Box) geom.Vec2 {
	var p geom.Vec2
	for x := b; x != nil; x = x.Parent {
		p = p.Add(x.Pos.Sub(x.Sticky))
	}
	return p
}

// stickyOffset is how far a sticky box moves from its place in the flow to keep its insets
// from the edges of its nearest scroll container's scrollport (the root's box when there
// is none), without leaving its parent's content box; a table cell's, the table's, its
// row and row group being no containing block.
func stickyOffset(b *box.Box) geom.Vec2 {
	parent := b.Parent
	if parent == nil {
		return geom.Vec2{}
	}
	sc := parent
	for sc.Parent != nil && (sc.Style == nil || !isScrollContainer(sc.Style)) {
		sc = sc.Parent
	}
	scAt := flowAt(sc)
	p := scrollport(sc)
	port := rect{l: scAt.X + p.X + sc.Scroll.X, t: scAt.Y + p.Y + sc.Scroll.Y}
	port.r, port.b = port.l+p.W, port.t+p.H

	if n := b.Node; n != nil && (n.Tag() == dom.Th || n.Tag() == dom.Td) {
		for parent.Parent != nil && parent.Node != nil &&
			(parent.Node.Tag() == dom.Tr || parent.Node.Tag() == dom.Thead || parent.Node.Tag() == dom.Tbody) {
			parent = parent.Parent
		}
	}
	pAt := flowAt(parent)
	cb := rect{
		l: pAt.X + parent.Border.Left + parent.Padding.Left,
		t: pAt.Y + parent.Border.Top + parent.Padding.Top,
		r: pAt.X + parent.Size.X - parent.Border.Right - parent.Padding.Right,
		b: pAt.Y + parent.Size.Y - parent.Border.Bottom - parent.Padding.Bottom,
	}
	if parent == sc && sc.Parent != nil {
		cb.r = pAt.X + p.X + max(sc.Content.X, p.W) - parent.Padding.Right
		cb.b = pAt.Y + p.Y + max(sc.Content.Y, p.H) - parent.Padding.Bottom
	}
	at := flowAt(b)
	el := rect{l: at.X, t: at.Y, r: at.X + b.Size.X, b: at.Y + b.Size.Y}
	m := b.Margin
	inset := b.Style.Inset
	axis := func(start, end, cbStart, cbEnd, marginStart, marginEnd, portStart,
		portEnd, portLen float32, insetStart, insetEnd style.Length) float32 {
		d := float32(0)
		if v := resolve(insetStart, portLen); some(v) && start < portStart+v {
			d = min(portStart+v-start, max(cbEnd-marginEnd-end, 0))
		}
		if v := resolve(insetEnd, portLen); some(v) && end+d > portEnd-v {
			d = max(portEnd-v-end, min(cbStart+marginStart-start, 0))
		}
		return d
	}
	return geom.Vec2{
		X: axis(el.l, el.r, cb.l, cb.r, m.Left, m.Right, port.l, port.r, p.W, inset.Left, inset.Right),
		Y: axis(el.t, el.b, cb.t, cb.b, m.Top, m.Bottom, port.t, port.b, p.H, inset.Top, inset.Bottom),
	}
}

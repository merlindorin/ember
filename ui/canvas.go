package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
)

// Canvas is a canvas element the game draws with draw, in its content box:
// c's origin is the box's top-left and c.Size() its size in points,
// clipped to it. Its width and height attributes are its natural size,
// 300 by 150 without them, which CSS sizes as an img's. draw runs when the
// canvas is first shown, after Invalidate, and when its box moves or
// resizes; otherwise its last drawing is kept, and never while it isn't
// shown. An ancestor's opacity below 1 doesn't fade what it draws yet.
// Exported for Chrome, it shows ember's own snapshot of its content box,
// which holds whatever ember painted over it there too.
func Canvas(draw func(c *twod.Canvas), args ...Arg) *Node {
	n := element(dom.Canvas, args)
	dom.Data(n).Canvas = &control.Canvas{Draw: draw}
	return n
}

// Invalidate has the canvas n drawn again on the next frame; it does
// nothing to anything else.
func (d *Document) Invalidate(n *Node) {
	if m, ok := d.machine(n).(*control.Canvas); ok {
		m.Invalidate()
		d.replay.invalidated = true
	}
}

// PointerOn is where the pointer is on the canvas n, from its content
// box's top-left in points; false while it isn't over the part of n that
// shows, or n isn't a canvas.
func (d *Document) PointerOn(n *Node) (geom.Vec2, bool) {
	if _, ok := d.machine(n).(*control.Canvas); !ok || len(d.hovered) == 0 || d.hovered[0] != n {
		return geom.Vec2{}, false
	}
	b := d.tree.Of[n]
	if b == nil {
		return geom.Vec2{}, false
	}
	r := n.Box()
	return d.pointerAt.Sub(geom.V2(r.X+b.Border.Left+b.Padding.Left, r.Y+b.Border.Top+b.Padding.Top)), true
}

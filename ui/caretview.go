package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
)

// caretState is where a text field's caret was and how long its text.
type caretState struct {
	n             *Node
	caret, length int
}

// followCaret scrolls the focused text field's own content, never its
// ancestors, so its caret shows, as Chrome does with block nearest: once
// the caret moved or the text changed, by keys, typing, a paste, a click
// or a drag.
func (d *Document) followCaret(size geom.Vec2) {
	n := d.focused()
	t := textOf(d.machine(n))
	if t == nil {
		d.caretSeen = caretState{}
		return
	}
	now := caretState{n: n, caret: t.Editor.Caret(), length: len(t.Editor.Runes())}
	if now == d.caretSeen {
		return
	}
	d.Layout(size)
	d.layText()
	b := d.tree.Of[n]
	if b == nil || t.Layout == nil {
		return
	}
	d.caretSeen = now
	at, ok := caretBox(b, t.Layout, now.caret)
	if !ok {
		return
	}
	port := b.ContentRect()
	off := b.Scroll
	if at.Y < port.Y+off.Y {
		off.Y = at.Y - port.Y
	} else if at.Y+at.H > port.Y+port.H+off.Y {
		off.Y = at.Y + at.H - port.Y - port.H
	}
	if at.X < port.X+off.X {
		off.X = at.X - port.X
	} else if at.X+at.W > port.X+port.W+off.X {
		off.X = at.X + at.W - port.X - port.W
	}
	off = geom.V2(max(off.X, 0), max(off.Y, 0))
	if off != b.Scroll {
		d.setScroll(b, off)
		d.replay.repaint = true
	}
}

// caretBox is the caret's bar in b's border box, before scrolling: a point
// wide, its line high.
func caretBox(b *box.Box, l interface {
	Line(int) int
	X(int) float32
}, i int) (geom.Rect, bool) {
	for _, c := range b.Children {
		if c.Kind != box.TextRun {
			continue
		}
		line := l.Line(i)
		var r geom.Rect
		if a, ok := l.(*area); ok {
			r = a.Row(line)
		} else if line < len(c.Lines) {
			r = c.Lines[line].Rect
		} else {
			return geom.Rect{}, false
		}
		return geom.Rect{X: c.Pos.X + r.X + l.X(i), Y: c.Pos.Y + r.Y, W: 1, H: r.H}, true
	}
	return geom.Rect{}, false
}

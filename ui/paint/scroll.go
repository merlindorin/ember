package paint

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// barSize is the room a bar takes along its edge and barWidth how wide it
// draws, as the old ember/ui's scrollbars.
const (
	barSize  = 10
	barWidth = 6
)

//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var thumbColor = color.RGBA(1, 1, 1, 0.25)

// Bars are where a scroll container's bars lie, in the root's space: the
// view they scroll, the tracks and thumbs of those showing, and the
// content's size.
type Bars struct {
	View                           geom.Rect
	VTrack, VThumb, HTrack, HThumb geom.Rect
	V, H                           bool
	Content                        geom.Vec2
	// Scroll is the offset the thumbs show.
	Scroll geom.Vec2
}

// A thumb is sized and placed on the whole bar, as Chrome sizes the box of
// export's bordered thumb, then drawn inset by thumbInset at both ends, as
// that box's borders inset what Chrome paints; minThumb is the least box.
const (
	thumbInset = 4
	minThumb   = 16 + 2*thumbInset
)

// BarsOf is where b's bars lie, its border box's top-left at at: along its
// padding box's right and bottom edges, in the gutter layout keeps beside
// the scrollport, overflow: scroll always and auto when the content
// overflows that way; none for scrollbar-width: none, which still scrolls
// by the wheel.
func BarsOf(b *box.Box, at geom.Vec2) Bars {
	view := padding(b, at)
	l := Bars{View: view, Content: b.Content, Scroll: b.Scroll}
	st := shown(b)
	o := st.Overflow
	l.V = !st.ScrollbarNone && (o.Height == style.Scroll || o.Height == style.OverflowAuto && b.Content.Y > view.H)
	l.H = !st.ScrollbarNone && (o.Width == style.Scroll || o.Width == style.OverflowAuto && b.Content.X > view.W)
	corner := func(on bool) float32 {
		if on {
			return barSize
		}
		return 0
	}
	if l.V {
		l.VTrack = geom.Rect{X: view.X + view.W - barSize + 2, Y: view.Y + 4, W: barWidth, H: view.H - 8 - corner(l.H)}
		if b.Content.Y > view.H {
			bar := view.H - corner(l.H)
			h := max(bar*view.H/b.Content.Y, minThumb)
			y := view.Y + (bar-h)*b.Scroll.Y/(b.Content.Y-view.H)
			l.VThumb = geom.Rect{X: l.VTrack.X, Y: y + thumbInset, W: barWidth, H: h - 2*thumbInset}
		}
	}
	if l.H {
		l.HTrack = geom.Rect{X: view.X + 4, Y: view.Y + view.H - barSize + 2, W: view.W - 8 - corner(l.V), H: barWidth}
		if b.Content.X > view.W {
			bar := view.W - corner(l.V)
			w := max(bar*view.W/b.Content.X, minThumb)
			x := view.X + (bar-w)*b.Scroll.X/(b.Content.X-view.W)
			l.HThumb = geom.Rect{X: x + thumbInset, Y: l.HTrack.Y, W: w - 2*thumbInset, H: barWidth}
		}
	}
	return l
}

func (l Bars) paint(s Surface, alpha float32) {
	c := fade(thumbColor, alpha)
	if l.VThumb.H > 0 {
		s.RoundRect(l.VThumb, 3, c)
	}
	if l.HThumb.W > 0 {
		s.RoundRect(l.HThumb, 3, c)
	}
}

// Over reports whether p is on a bar showing.
func (l Bars) Over(p geom.Vec2) bool {
	return l.V && l.VTrack.Contains(p) || l.H && l.HTrack.Contains(p)
}

// Pointer is the mouse over a scroll container this frame.
type Pointer struct {
	At geom.Vec2
	// Pressed is the left button going down this frame, Down it held.
	Pressed, Down bool
	// Wheel is the wheel's lines this frame; Shift turns a vertical wheel
	// sideways.
	Wheel geom.Vec2
	Shift bool
	// Hover is the pointer over the container, nothing covering it.
	Hover bool
}

// Scrolling is a scroll container's drag of its thumbs, kept between
// frames by its node.
type Scrolling struct {
	dragV, dragH bool
	grab         float32
}

// Input is the offset l's container scrolls to for p, step points a wheel
// line, as the old ember/ui's bars do: the wheel turns it, a thumb drags it, a
// click on a track pages it. It reports whether the bars took the mouse,
// so what is under them doesn't also take the click.
func (s *Scrolling) Input(l Bars, p Pointer, step float32) (geom.Vec2, bool) {
	offset, took := l.Scroll, false
	if w := p.Wheel; p.Hover && (w.X != 0 || w.Y != 0) {
		if p.Shift && w.X == 0 {
			w = geom.V2(w.Y, 0)
		}
		offset = offset.Add(w.Scale(step * 3))
	}
	if p.Pressed {
		switch {
		case l.V && l.VTrack.Contains(p.At):
			took = true
			if l.VThumb.Contains(p.At) {
				s.dragV, s.grab = true, p.At.Y-l.VThumb.Y
			} else if l.VThumb.H > 0 {
				offset.Y += page(p.At.Y < l.VThumb.Y, l.View.H)
			}
		case l.H && l.HTrack.Contains(p.At):
			took = true
			if l.HThumb.Contains(p.At) {
				s.dragH, s.grab = true, p.At.X-l.HThumb.X
			} else if l.HThumb.W > 0 {
				offset.X += page(p.At.X < l.HThumb.X, l.View.W)
			}
		}
	}
	switch {
	case !p.Down:
		s.dragV, s.dragH = false, false
	case s.dragV:
		if travel := l.VTrack.H - l.VThumb.H; travel > 0 {
			offset.Y = (p.At.Y - s.grab - l.VTrack.Y) / travel * (l.Content.Y - l.View.H)
		}
		took = true
	case s.dragH:
		if travel := l.HTrack.W - l.HThumb.W; travel > 0 {
			offset.X = (p.At.X - s.grab - l.HTrack.X) / travel * (l.Content.X - l.View.W)
		}
		took = true
	}
	offset.X = min(max(offset.X, 0), max(l.Content.X-l.View.W, 0))
	offset.Y = min(max(offset.Y, 0), max(l.Content.Y-l.View.H, 0))
	return offset, took
}

// Dragging reports whether a thumb is being dragged.
func (s *Scrolling) Dragging() bool { return s.dragV || s.dragH }

func page(back bool, by float32) float32 {
	if back {
		return -by
	}
	return by
}

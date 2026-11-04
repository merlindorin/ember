package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

// list is a 100×100 scroller of 400 points of content, scrolled to y.
func list(y float32) *box.Box {
	b := el(0, 0, 100, 100, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.OverflowAuto}
	})
	b.Content, b.Scroll = geom.V2(100, 400), geom.V2(0, y)
	return b
}

func TestTheWheelScrollsThreeLinesAndShiftTurnsItSideways(t *testing.T) {
	var s Scrolling
	l := BarsOf(list(0), geom.Vec2{})
	off, took := s.Input(l, Pointer{At: geom.V2(50, 50), Hover: true, Wheel: geom.V2(0, 2)}, 10)
	assert.Equal(t, geom.V2(0, 60), off)
	assert.False(t, took)
	off, _ = s.Input(l, Pointer{At: geom.V2(50, 50), Wheel: geom.V2(0, 2)}, 10)
	assert.Equal(t, geom.Vec2{}, off, "only while hovered")

	wide := el(0, 0, 100, 100, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.Hidden}
	})
	wide.Content = geom.V2(400, 100)
	off, _ = s.Input(BarsOf(wide, geom.Vec2{}), Pointer{Hover: true, Shift: true, Wheel: geom.V2(0, 1)}, 10)
	assert.Equal(t, geom.V2(30, 0), off)
}

func TestTheOffsetStaysWithinTheContent(t *testing.T) {
	var s Scrolling
	off, _ := s.Input(BarsOf(list(290), geom.Vec2{}), Pointer{Hover: true, Wheel: geom.V2(0, 5)}, 10)
	assert.Equal(t, geom.V2(0, 300), off)
	off, _ = s.Input(BarsOf(list(10), geom.Vec2{}), Pointer{Hover: true, Wheel: geom.V2(0, -5)}, 10)
	assert.Equal(t, geom.Vec2{}, off)
}

func TestAClickOnTheTrackPagesByTheView(t *testing.T) {
	var s Scrolling
	l := BarsOf(list(0), geom.Vec2{})
	below := geom.V2(l.VTrack.Center().X, l.VTrack.Y+l.VTrack.H-2)
	off, took := s.Input(l, Pointer{At: below, Pressed: true, Down: true, Hover: true}, 10)
	assert.True(t, took, "the bar takes the click from what is under it")
	assert.Equal(t, geom.V2(0, 100), off)
	assert.False(t, s.Dragging())
}

func TestDraggingTheThumbFollowsThePointer(t *testing.T) {
	var s Scrolling
	b := list(0)
	l := BarsOf(b, geom.Vec2{})
	grab := l.VThumb.Center()
	_, took := s.Input(l, Pointer{At: grab, Pressed: true, Down: true, Hover: true}, 10)
	assert.True(t, took)
	assert.True(t, s.Dragging())
	travel := l.VTrack.H - l.VThumb.H
	off, _ := s.Input(l, Pointer{At: grab.Add(geom.V2(30, travel/2)), Down: true}, 10)
	assert.InDelta(t, 150, off.Y, 1e-3, "half the travel is half the overflow, wherever the pointer is aside")
	s.Input(l, Pointer{At: grab}, 10)
	assert.False(t, s.Dragging(), "released")
}

func TestScrollAlwaysShowsATrackButAThumbOnlyWhenItOverflows(t *testing.T) {
	b := el(0, 0, 100, 100, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Scroll, Height: style.Scroll}
	})
	b.Content = geom.V2(100, 100)
	l := BarsOf(b, geom.Vec2{})
	assert.True(t, l.V)
	assert.True(t, l.H)
	assert.Equal(t, geom.Rect{}, l.VThumb)
	assert.Equal(t, float32(100-8-barSize), l.VTrack.H, "the corner is left to the other bar")
	assert.True(t, l.Over(l.VTrack.Center()))
}

func TestAScrollersContentIsClippedToItsScrollportBesideTheBar(t *testing.T) {
	b := list(0)
	b.Scrollbar = geom.V2(15, 0)
	b.Append(el(0, 0, 100, 400, bg(red)))
	r := paintOne(b)
	var clip geom.Rect
	for _, o := range r.of("rect") {
		if o.Color == red {
			clip = o.Clip
		}
	}
	assert.Equal(t, geom.Rect{W: 85, H: 100}, clip, "the padding box less the bar's gutter, as Chrome's scrollport")
}

func TestScrollbarWidthNoneDrawsNoBars(t *testing.T) {
	b := list(0)
	b.Style.ScrollbarNone = true
	l := BarsOf(b, geom.Vec2{})
	assert.False(t, l.V)
	assert.False(t, l.H)
	var s Scrolling
	off, _ := s.Input(l, Pointer{At: geom.V2(50, 50), Hover: true, Wheel: geom.V2(0, 2)}, 10)
	assert.Equal(t, geom.V2(0, 60), off, "the wheel still scrolls it")
}

func TestAThumbIsSizedOnTheWholeBarAndInsetByItsEnds(t *testing.T) {
	top, bottom := BarsOf(list(0), geom.Vec2{}), BarsOf(list(300), geom.Vec2{})
	assert.Equal(t, float32(25-8), top.VThumb.H,
		"a quarter of the bar's 100, less 4 at each end, as Chrome's bordered thumb")
	assert.Equal(t, float32(4), top.VThumb.Y, "at the top, its border's 4 in")
	assert.Equal(t, float32(100-25+4), bottom.VThumb.Y, "at the bottom, the box's 75 down plus 4")
}

package ui_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scroller is a page with a list 200×100 of ten 40-high buttons, inside
// an outer box 150 high followed by 300 points more, so both scroll.
type scroller struct {
	doc         *ui.Document
	w           *world.World
	in          *input.State
	outer, list *ui.Node
	items       []*ui.Node
}

func newScroller(t *testing.T) *scroller {
	t.Helper()
	s := &scroller{}
	s.list = ui.Div(ui.ID("list"), ui.Style("overflow: auto; width: 200px; height: 100px"))
	for i := range 10 {
		b := ui.Button(ui.Style("display: block; width: 180px; height: 40px"), ui.Text(fmt.Sprint("item ", i)))
		s.items = append(s.items, b)
		s.list.Append(b)
	}
	s.outer = ui.Div(ui.ID("outer"), ui.Style("overflow: auto; width: 240px; height: 150px"),
		s.list, ui.Div(ui.Style("height: 300px")))
	s.doc = ui.NewDocument()
	s.doc.Body().Append(s.outer)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(s.doc)
	s.w = ember.New().AddEntities(win)
	s.in = win.Input()
	s.w.RunFrames(2)
	return s
}

// scrolled is how far the list's content has moved up.
func (s *scroller) scrolled() float32 { return s.list.Box().Y - s.items[0].Box().Y }

func (s *scroller) outerScrolled() float32 { return s.outer.Box().Y - s.list.Box().Y }

func (s *scroller) wheel(at geom.Vec2, d geom.Vec2) {
	s.in.SetMouse(at)
	s.w.RunFrames(1)
	s.in.AddWheel(d)
	s.w.RunFrames(2)
}

func (s *scroller) key(k input.Key) {
	s.in.SetKey(k, true)
	s.w.RunFrames(1)
	s.in.SetKey(k, false)
	s.w.RunFrames(1)
}

func TestTheWheelScrollsTheScrollerUnderThePointer(t *testing.T) {
	s := newScroller(t)
	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	assert.Positive(t, s.scrolled(), "a notch down")
	assert.Zero(t, s.outerScrolled(), "the inner one took it")
	first := s.scrolled()
	s.wheel(s.list.Box().Center(), geom.V2(0, -1))
	assert.InDelta(t, 0, s.scrolled(), 1e-3, "and back")
	assert.Greater(t, first, float32(20), "three lines a notch, as the old ember/ui")
}

func TestAtItsEndTheWheelScrollsTheNextScrollerOut(t *testing.T) {
	s := newScroller(t)
	for range 20 {
		s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	}
	assert.InDelta(t, 300, s.scrolled(), 1e-3, "the list at its end")
	assert.Positive(t, s.outerScrolled(), "the outer box took the rest")
}

func TestShiftTurnsTheWheelSideways(t *testing.T) {
	wide := ui.Div(ui.Style("width: 600px; height: 20px"))
	box := ui.Div(ui.Style("overflow: auto; width: 200px; height: 60px"), wide)
	doc := ui.NewDocument()
	doc.Body().Append(box)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	in := win.Input()
	w.RunFrames(2)
	in.SetMouse(box.Box().Center())
	in.SetKey(input.KeyLeftShift, true)
	w.RunFrames(1)
	in.AddWheel(geom.V2(0, 1))
	w.RunFrames(2)
	in.SetKey(input.KeyLeftShift, false)
	assert.Positive(t, box.Box().X-wide.Box().X)
}

func TestDraggingTheThumbScrollsAndClicksNothing(t *testing.T) {
	s := newScroller(t)
	l := s.list.Box()
	grab := geom.V2(l.X+l.W-10+2+3, l.Y+4+5)
	s.in.SetMouse(grab)
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.w.RunFrames(1)
	s.in.SetMouse(grab.Add(geom.V2(0, 30)))
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.w.RunFrames(2)
	travel := float32(100 - 25)
	assert.InDelta(t, 30/travel*300, s.scrolled(), 1, "the thumb's travel is the overflow's")
	var r world.Reader[ui.Clicked]
	assert.Zero(t, r.Count(s.w), "the bar took the press")
}

func TestAClickOnTheTrackPagesByTheView(t *testing.T) {
	s := newScroller(t)
	l := s.list.Box()
	s.in.SetMouse(geom.V2(l.X+l.W-10+2+3, l.Y+l.H-8))
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.w.RunFrames(2)
	assert.InDelta(t, 100, s.scrolled(), 1e-3)
}

func TestPageAndEndKeysScrollTheFocusedElementsScroller(t *testing.T) {
	s := newScroller(t)
	s.doc.Focus(s.items[0])
	s.w.RunFrames(1)
	require.Zero(t, s.scrolled())
	s.key(input.KeyPageDown)
	assert.InDelta(t, 100, s.scrolled(), 1e-3, "a page is the view")
	s.key(input.KeyEnd)
	assert.InDelta(t, 300, s.scrolled(), 1e-3)
	s.key(input.KeyHome)
	assert.InDelta(t, 0, s.scrolled(), 1e-3)
}

func TestFocusByKeyboardScrollsTheElementIntoViewByItsNearestEdge(t *testing.T) {
	s := newScroller(t)
	s.doc.Focus(s.items[0])
	s.w.RunFrames(1)
	for range 3 {
		s.key(input.KeyTab)
	}
	l, b := s.list.Box(), s.items[3].Box()
	assert.InDelta(t, l.Y+l.H, b.Y+b.H, 1e-3, "item 3 just shows at the bottom")
	for range 3 {
		s.key(input.KeyTab)
	}
	l, b = s.list.Box(), s.items[6].Box()
	assert.InDelta(t, l.Y+l.H, b.Y+b.H, 1e-3)
	s.in.SetKey(input.KeyLeftShift, true)
	for range 6 {
		s.key(input.KeyTab)
	}
	s.in.SetKey(input.KeyLeftShift, false)
	l, b = s.list.Box(), s.items[0].Box()
	assert.InDelta(t, l.Y, b.Y, 1e-3, "back up: item 0 at the top")
}

func TestWithNothingFocusedPageKeysScrollWhatIsUnderThePointer(t *testing.T) {
	s := newScroller(t)
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.key(input.KeyPageDown)
	assert.InDelta(t, 100, s.scrolled(), 1e-3, "the list under the pointer")
	s.key(input.KeyEnd)
	assert.InDelta(t, 300, s.scrolled(), 1e-3)

	o := s.outer.Box()
	s.in.SetMouse(geom.V2(o.X+o.W-20, o.Y+o.H-10))
	s.w.RunFrames(1)
	s.key(input.KeyPageDown)
	assert.Positive(t, s.outerScrolled(), "beside the list: the outer box")
}

func TestAFocusedScrollContainerPagesItself(t *testing.T) {
	s := newScroller(t)
	s.list.SetAttr("tabindex", "0")
	s.w.RunFrames(1)
	s.doc.Focus(s.list)
	s.w.RunFrames(1)
	s.key(input.KeyPageDown)
	assert.InDelta(t, 100, s.scrolled(), 1e-3)
}

func TestAPagingKeyIsTheGamesOnlyWhenNothingScrolled(t *testing.T) {
	s := newScroller(t)
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.in.SetKey(input.KeyPageDown, true)
	s.w.RunFrames(1)
	assert.False(t, s.in.Down(input.KeyPageDown), "it scrolled the list: the game doesn't see it")
	s.in.SetKey(input.KeyPageDown, false)
	s.w.RunFrames(2)

	s.in.SetMouse(geom.V2(390, 290))
	s.w.RunFrames(1)
	s.in.SetKey(input.KeyHome, true)
	s.w.RunFrames(1)
	assert.True(t, s.in.Down(input.KeyHome), "nothing there to scroll home: the game has it")
	s.in.SetKey(input.KeyHome, false)
}

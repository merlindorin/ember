package ui_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
)

const windowSheet = `
dialog.w { position: fixed; left: 40px; top: 30px; width: 200px; height: 120px; margin: 0; padding: 0;
	border: 1px solid #000; min-width: 120px; min-height: 60px; }
dialog.w[data-zoomed] { left: 0; top: 0; width: auto; height: auto; right: 0; bottom: 0; }
.bar { display: block; height: 20px; }
.bar > button { width: 10px; height: 10px; padding: 0; border: 0; }
`

// win is a window as kit builds one: a dialog holding a bar with the three lights, then a
// body.
func win(extra ...ui.Arg) (*ui.Node, *ui.Node, map[string]*ui.Node) {
	lights := map[string]*ui.Node{}
	bar := ui.Header(ui.Class("bar"), ui.Attr("data-window-bar", ""))
	for _, a := range []string{"close", "minimise", "zoom"} {
		lights[a] = ui.Button(ui.Attr("data-window-action", a), ui.Attr("aria-label", a))
		bar.Append(lights[a])
	}
	w := ui.Dialog(append([]ui.Arg{ui.Class("w"), bar, ui.Div(ui.Text("body"))}, extra...)...)
	return w, bar, lights
}

func windowDoc(t *testing.T, nodes ...*ui.Node) *live {
	t.Helper()
	l := &live{doc: ui.NewDocument(ui.Stylesheet(fstest.MapFS{"w.css": {Data: []byte(windowSheet)}}, "w.css")),
		win: platform.NewWindow(platform.Size(640, 360))}
	l.doc.Body().Append(nodes...)
	l.win.Add(l.doc)
	l.auto = automation.New(automation.Recording())
	l.w = ember.New(ember.Automation(l.auto)).AddEntities(l.win)
	for _, n := range nodes {
		l.doc.Show(n)
	}
	l.frames(2)
	return l
}

// drag presses at from, moves to to, and releases.
func (l *live) drag(from, to geom.Vec2) {
	in := l.win.Input()
	in.SetMouse(from)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouse(to)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(2)
}

func styleOf(n *ui.Node, property string) string {
	for _, d := range strings.Split(n.Style(), ";") {
		if k, v, ok := strings.Cut(d, ":"); ok && strings.TrimSpace(k) == property {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func TestAWindowIsDraggedByItsBar(t *testing.T) {
	w, bar, _ := win()
	l := windowDoc(t, w)
	b := bar.Box()
	l.drag(geom.V2(b.X+100, b.Y+10), geom.V2(b.X+150, b.Y+70))
	assert.Equal(t, "90px", styleOf(w, "left"))
	assert.Equal(t, "90px", styleOf(w, "top"))
	assert.InDelta(t, 90, w.Box().X, 0.01, "laid out where it was dropped")
}

func TestADragKeepsTheBarInTheWindow(t *testing.T) {
	w, bar, _ := win()
	l := windowDoc(t, w)
	b := bar.Box()
	l.drag(geom.V2(b.X+100, b.Y+10), geom.V2(b.X+100, b.Y-200))
	assert.Equal(t, "0px", styleOf(w, "top"), "the bar never goes above the window")
	l.drag(geom.V2(w.Box().X+100, w.Box().Y+10), geom.V2(900, 600))
	assert.LessOrEqual(t, w.Box().X, float32(640-20), "at least a bar's height of it stays in")
	assert.LessOrEqual(t, w.Box().Y, float32(360-20))
}

func TestAZoomedWindowIsNotDragged(t *testing.T) {
	w, _, lights := win()
	l := windowDoc(t, w)
	l.clickAt(lights["zoom"])
	require.Equal(t, float32(0), w.Box().X)
	l.drag(geom.V2(100, 10), geom.V2(200, 100))
	assert.Equal(t, float32(0), w.Box().X)
	assert.Empty(t, styleOf(w, "left"))
}

func TestAResizableWindowIsResizedByItsEdgesWithinItsMinimum(t *testing.T) {
	w, _, _ := win(ui.Attr("data-resizable", ""))
	l := windowDoc(t, w)
	r := w.Box()
	l.drag(geom.V2(r.X+r.W-2, r.Y+r.H-2), geom.V2(r.X+r.W+48, r.Y+r.H+28))
	assert.InDelta(t, 250, w.Box().W, 0.01, "the corner resizes both ways")
	assert.InDelta(t, 150, w.Box().H, 0.01)
	r = w.Box()
	l.drag(geom.V2(r.X+r.W-2, r.Y+r.H/2), geom.V2(r.X, r.Y+r.H/2))
	assert.InDelta(t, 120, w.Box().W, 0.01, "no narrower than min-width")
	assert.InDelta(t, 150, w.Box().H, 0.01, "the right edge leaves the height")
}

func TestTheCursorShowsHowAnEdgeResizes(t *testing.T) {
	w, _, _ := win(ui.Attr("data-resizable", ""))
	l := windowDoc(t, w)
	r := w.Box()
	for at, want := range map[geom.Vec2]platform.Cursor{
		geom.V2(r.X+r.W-2, r.Y+r.H/2): platform.CursorResizeEW,
		geom.V2(r.X+r.W/2, r.Y+r.H-2): platform.CursorResizeNS,
		geom.V2(r.X+r.W-2, r.Y+r.H-2): platform.CursorResizeNWSE,
		geom.V2(r.X+r.W/2, r.Y+r.H/2): platform.CursorArrow,
	} {
		l.win.Input().SetMouse(at)
		l.frames(2)
		assert.Equal(t, want, l.win.Cursor(), "at %v", at)
	}
}

func TestAPressRaisesAWindowOverTheOthers(t *testing.T) {
	a, _, _ := win()
	b, _, _ := win(ui.Style("left: 300px"))
	l := windowDoc(t, a, b)
	l.clickAt(a.Children()[1])
	za := styleOf(a, "z-index")
	require.NotEmpty(t, za)
	l.clickAt(b.Children()[1])
	assert.Greater(t, styleOf(b, "z-index"), za)
	assert.Same(t, a, a.Parent().Children()[0], "raising never moves the node")
}

func TestTheCloseLightClosesTheWindow(t *testing.T) {
	w, _, lights := win()
	l := windowDoc(t, w)
	l.clickAt(lights["close"])
	assert.False(t, l.doc.Open(w))
	require.Len(t, l.closed, 1)
	assert.Same(t, w, l.closed[0].Node)
	assert.Contains(t, l.clicks, lights["close"])
}

func TestTheMinimiseLightFoldsTheWindow(t *testing.T) {
	w, _, lights := win()
	l := windowDoc(t, w)
	l.clickAt(lights["minimise"])
	assert.True(t, hasAttr(w, "data-minimised"))
	assert.Contains(t, l.clicks, lights["minimise"], "the light's Clicked still fires")
	l.clickAt(lights["minimise"])
	assert.False(t, hasAttr(w, "data-minimised"))
}

func TestTheZoomLightFillsTheWindowOverMinimised(t *testing.T) {
	w, _, lights := win()
	l := windowDoc(t, w)
	l.clickAt(lights["minimise"])
	l.clickAt(lights["zoom"])
	assert.True(t, hasAttr(w, "data-zoomed"))
	assert.False(t, hasAttr(w, "data-minimised"), "zoomed wins over minimised")
	assert.Equal(t, geom.V2(640, 360), geom.V2(w.Box().W, w.Box().H))
	assert.Contains(t, l.clicks, lights["zoom"])
	assert.Contains(t, l.roles(automation.RoleButton), "zoom", "the lights are named for automation")
}

func hasAttr(n *ui.Node, name string) bool { _, ok := n.Attr(name); return ok }

func TestACSSCursorShowsOverAnyElement(t *testing.T) {
	link := ui.Div(ui.Style("width: 80px; height: 20px; cursor: pointer"))
	plain := ui.Div(ui.Style("width: 80px; height: 20px"))
	l := liveDoc(t, link, plain)
	l.win.Input().SetMouse(link.Box().Center())
	l.frames(2)
	assert.Equal(t, platform.CursorHand, l.win.Cursor())
	l.win.Input().SetMouse(plain.Box().Center())
	l.frames(2)
	assert.Equal(t, platform.CursorArrow, l.win.Cursor())
}

func TestAPressOnAControlInTheBarClicksItRatherThanDragging(t *testing.T) {
	w, bar, _ := win()
	back := ui.Button(ui.Text("Back"), ui.Style("width: 40px; height: 16px"))
	bar.Append(back)
	l := windowDoc(t, w)
	b := back.Box()
	l.drag(b.Center(), b.Center().Add(geom.V2(30, 30)))
	assert.Empty(t, styleOf(w, "left"), "no drag from a button in the bar")
	l.clickAt(back)
	assert.Contains(t, l.clicks, back)
}

func TestADragTellsWhereTheWindowWasMovedOnceReleased(t *testing.T) {
	w, bar, _ := win()
	l := windowDoc(t, w)
	b := bar.Box()
	l.clickAt(bar)
	assert.Empty(t, l.windows, "a press on its bar alone moves nothing")
	assert.False(t, l.doc.MovedByUser(w))
	l.drag(geom.V2(b.X+100, b.Y+10), geom.V2(b.X+150, b.Y+70))
	assert.Equal(t, []any{ui.WindowMoved{Node: w, At: geom.V2(90, 90)}}, l.windows, "once, where it was dropped")
	assert.True(t, l.doc.MovedByUser(w))
}

func TestAnEdgeDragTellsTheWindowsNewSize(t *testing.T) {
	w, _, _ := win(ui.Attr("data-resizable", ""))
	l := windowDoc(t, w)
	r := w.Box()
	l.drag(geom.V2(r.X+r.W-2, r.Y+r.H-2), geom.V2(r.X+r.W+48, r.Y+r.H+28))
	require.Len(t, l.windows, 1)
	e, ok := l.windows[0].(ui.WindowResized)
	require.True(t, ok, "%T", l.windows[0])
	assert.Same(t, w, e.Node)
	assert.InDelta(t, 250, e.Size.X, 0.01)
	assert.InDelta(t, 150, e.Size.Y, 0.01)
}

func TestAnEdgeDragTellsTheSizeTheWindowTookWithinItsLimits(t *testing.T) {
	w, _, _ := win(ui.Attr("data-resizable", ""), ui.Style("max-width: 220px; max-height: 130px"))
	l := windowDoc(t, w)
	r := w.Box()
	l.drag(geom.V2(r.X+r.W-2, r.Y+r.H-2), geom.V2(r.X+r.W+148, r.Y+r.H+128))
	require.Len(t, l.windows, 1)
	e, ok := l.windows[0].(ui.WindowResized)
	require.True(t, ok, "%T", l.windows[0])
	b := w.Box()
	assert.InDelta(t, b.W, e.Size.X, 0.01, "the width it took")
	assert.InDelta(t, b.H, e.Size.Y, 0.01, "the height it took")
	assert.LessOrEqual(t, e.Size.Y, float32(130))
}

func TestTheMinimiseLightTellsTheWindowFoldedAndUnfolded(t *testing.T) {
	w, _, lights := win()
	l := windowDoc(t, w)
	l.clickAt(lights["minimise"])
	l.clickAt(lights["minimise"])
	assert.Equal(t, []any{ui.WindowFolded{Node: w, Folded: true}, ui.WindowFolded{Node: w, Folded: false}}, l.windows)
}

func TestAWindowIsPlacedSizedAndFoldedByCodeWithoutEvents(t *testing.T) {
	w, _, _ := win()
	l := windowDoc(t, w)
	l.doc.PlaceWindow(w, geom.V2(300, 100))
	l.doc.SizeWindow(w, geom.V2(160, 90))
	l.doc.SetFolded(w, true)
	l.frames(2)
	assert.InDelta(t, 300, w.Box().X, 0.01)
	assert.InDelta(t, 100, w.Box().Y, 0.01)
	assert.InDelta(t, 160, w.Box().W, 0.01)
	assert.True(t, hasAttr(w, "data-minimised"))
	assert.Empty(t, l.windows, "as SetChecked, no events")
	l.doc.SizeWindow(w, geom.V2(180, 0))
	l.frames(1)
	assert.InDelta(t, 180, w.Box().W, 0.01)
	assert.Equal(t, "90px", styleOf(w, "height"), "a side of 0 left as it was")
	assert.False(t, l.doc.MovedByUser(w), "placed by the game, not moved by the user")
}

func TestOnFoldRunsWhenTheWindowFolds(t *testing.T) {
	var got []ui.Event
	w, _, lights := win(ui.OnFold(func(e ui.Event) { got = append(got, e) }))
	l := windowDoc(t, w)
	l.clickAt(lights["minimise"])
	require.Len(t, got, 1)
	assert.Equal(t, ui.EventFold, got[0].Type)
	assert.True(t, got[0].Folded)
	assert.Same(t, w, got[0].Node)
}

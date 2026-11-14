package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/world"
)

// owned is a window with a page Document and, drawn after it, a
// Document holding a modal dialog.
type owned struct {
	w           *world.World
	win         *platform.Window
	page, top   *Document
	behind      *Node
	field       *Node
	dlg, ok     *Node
	clicks      []*Node
	gameClicked []bool
}

func newOwned(t *testing.T) *owned {
	t.Helper()
	o := &owned{page: NewDocument(), top: NewDocument(), win: platform.NewWindow(platform.Size(640, 360))}
	o.behind = Button(Style("position: absolute; left: 10px; top: 10px; width: 120px; height: 30px"), Text("Behind"))
	o.field = Input(Style("position: absolute; left: 10px; top: 60px"))
	o.page.Body().Append(o.behind, o.field)
	o.ok = Button(Text("OK"))
	o.dlg = Dialog(Style("left: 400px; top: 200px; margin: 0"), o.ok)
	o.top.Body().Append(o.dlg)
	o.win.Add(o.page, o.top)
	o.w = ember.New().AddEntities(o.win)
	o.w.AddSystems(&gameClick{o: o})
	o.frames(2)
	return o
}

type gameClick struct{ o *owned }

func (g *gameClick) Update(world.Frame) {
	in := g.o.win.Input()
	if in.Pressed(input.MouseLeft) {
		g.o.gameClicked = append(g.o.gameClicked, !in.PointerCaptured())
	}
}

func (o *owned) frames(n int) {
	for range n {
		o.w.RunFrames(1)
		var r world.Reader[Clicked]
		for e := range r.Read(o.w) {
			o.clicks = append(o.clicks, e.Node)
		}
	}
}

func (o *owned) click(at geom.Vec2) {
	in := o.win.Input()
	in.SetMouse(at)
	o.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	o.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	o.frames(2)
}

func TestAModalOwnsItsWindowsInputOverAnotherDocument(t *testing.T) {
	o := newOwned(t)
	o.page.Focus(o.field)
	o.frames(1)
	o.top.ShowModal(o.dlg)
	o.frames(2)
	require.True(t, o.top.Open(o.dlg))

	o.win.Input().SetMouse(o.behind.Box().Center())
	o.frames(2)
	assert.False(t, o.behind.State().Has(dom.Hover), "the page under a modal of another document hovers nothing")
	o.win.Input().AddText('x')
	o.frames(2)
	assert.Empty(t, o.page.Value(o.field), "it gets no keys, even with its focus")
	o.click(o.behind.Box().Center())
	assert.NotContains(t, o.clicks, o.behind, "nor clicks")

	o.click(o.ok.Box().Center())
	assert.Contains(t, o.clicks, o.ok, "the modal's document takes them")

	o.top.Close(o.dlg, "")
	o.page.Focus(o.field)
	o.frames(1)
	o.win.Input().AddText('y')
	o.frames(2)
	assert.Equal(t, "y", o.page.Value(o.field), "closed, the page gets its keys back")
}

func TestAGameSystemsClickUnderAModalReadsAsCaptured(t *testing.T) {
	o := newOwned(t)
	o.click(geom.V2(300, 300))
	require.Equal(t, []bool{true}, o.gameClicked, "no modal: the game's click")
	o.top.ShowModal(o.dlg)
	o.win.Input().SetMouseButton(input.MouseLeft, true)
	o.frames(1)
	assert.Equal(t, []bool{true, false}, o.gameClicked, "a click the frame a modal opens, before it draws, is captured")
}

// spriteClicks counts the clicks a game takes on its own 2D entity: a
// press inside rect that no Document captured.
type spriteClicks struct {
	win  *platform.Window
	rect geom.Rect
	n    int
}

func (s *spriteClicks) Update(world.Frame) {
	in := s.win.Input()
	if in.Pressed(input.MouseLeft) && !in.PointerCaptured() && s.rect.Contains(in.Mouse()) {
		s.n++
	}
}

func TestAGames2DEntityUnderAModalGetsNoClick(t *testing.T) {
	o := newOwned(t)
	game := &spriteClicks{win: o.win, rect: geom.Rect{X: 10, Y: 280, W: 100, H: 40}}
	o.w.AddSystems(game)
	o.top.ShowModal(o.dlg)
	o.frames(2)
	o.click(game.rect.Center())
	assert.Zero(t, game.n, "the game's entity under the modal gets no click")

	o.top.Close(o.dlg, "")
	o.click(game.rect.Center())
	assert.Equal(t, 1, game.n, "closed, it takes clicks again")
}

func TestClosingAModalHandsTheInputBackAtOnce(t *testing.T) {
	o := newOwned(t)
	o.top.ShowModal(o.dlg)
	assert.Equal(t, wm.Entity(o.top), wm.Owner(o.w, o.win), "showing a modal owns the window at once")
	o.frames(2)
	o.top.Close(o.dlg, "")
	assert.Nil(t, wm.Owner(o.w, o.win), "closing it hands the window back at once")
	o.win.Input().SetMouse(o.behind.Box().Center())
	o.win.Input().SetMouseButton(input.MouseLeft, true)
	o.frames(1)
	assert.True(t, o.behind.State().Has(dom.Active), "the page takes the press the same frame")
}

func TestModalOwnersStackTheLastOpenerWins(t *testing.T) {
	o := newOwned(t)
	second := Dialog(Button(Text("Inner")))
	o.page.Body().Append(second)
	o.frames(1)
	o.top.ShowModal(o.dlg)
	o.page.ShowModal(second)
	assert.Equal(t, wm.Entity(o.page), wm.Owner(o.w, o.win), "the last modal opened owns the window")
	o.page.Close(second, "")
	assert.Equal(t, wm.Entity(o.top), wm.Owner(o.w, o.win), "closing it hands the window to the modal below")
	o.top.Close(o.dlg, "")
	assert.Nil(t, wm.Owner(o.w, o.win))
}

func TestAHiddenDocumentGivesItsWindowBackUntilShown(t *testing.T) {
	o := newOwned(t)
	o.top.ShowModal(o.dlg)
	o.frames(2)
	o.top.SetHidden(true)
	o.frames(2)
	assert.Nil(t, wm.Owner(o.w, o.win), "hidden with its modal open, the document owns nothing")
	assert.False(t, o.win.Input().PointerCaptured(), "and the game has the window back")
	o.click(o.behind.Box().Center())
	assert.Contains(t, o.clicks, o.behind, "as has the page")

	o.top.SetHidden(false)
	o.frames(1)
	assert.Equal(t, wm.Entity(o.top), wm.Owner(o.w, o.win), "shown again, its open modal owns the window again")
}

func TestAUIElementOverAGames2DEntityTakesItsClick(t *testing.T) {
	doc := NewDocument()
	cover := Div(Style("position: absolute; left: 0; top: 280px; width: 300px; height: 80px"))
	doc.Body().Append(cover)
	win := platform.NewWindow(platform.Size(640, 360))
	win.Add(doc)
	game := &spriteClicks{win: win, rect: geom.Rect{X: 10, Y: 290, W: 100, H: 40}}
	w := ember.New().AddEntities(win).AddSystems(game)
	w.RunFrames(2)
	click := func() {
		win.Input().SetMouse(game.rect.Center())
		w.RunFrames(1)
		win.Input().SetMouseButton(input.MouseLeft, true)
		w.RunFrames(1)
		win.Input().SetMouseButton(input.MouseLeft, false)
		w.RunFrames(2)
	}
	click()
	assert.Zero(t, game.n, "a ui element drawn over the entity takes the click")

	cover.SetStyle("position: absolute; left: 0; top: 280px; width: 300px; height: 80px; pointer-events: none")
	w.RunFrames(2)
	click()
	assert.Equal(t, 1, game.n, "with pointer-events: none it lets the click through to the game")
}

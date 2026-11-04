package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// glide moves the pointer from a to b a point a frame, as a hand or a
// scripted capture does, and reports the first point where open turned
// false, if any.
func glide(s *shown, a, b geom.Vec2, open func() bool) (geom.Vec2, bool) {
	steps := int(max(abs(b.X-a.X), abs(b.Y-a.Y)))
	for i := 1; i <= steps; i++ {
		p := a.Add(b.Sub(a).Scale(float32(i) / float32(steps)))
		s.in.SetMouse(p)
		s.frames(1)
		if !open() {
			return p, true
		}
	}
	return geom.Vec2{}, false
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func pressHere(s *shown) {
	s.in.SetMouseButton(input.MouseLeft, true)
	s.frames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.frames(2)
}

func TestAMenuStaysOpenWhileThePointerWalksDownFromItsTitleToAnItem(t *testing.T) {
	ran := 0
	export := kit.MenuItem("Export", ui.OnClick(func() { ran++ }))
	fileMenu := kit.Menu(export, kit.MenuItem("Import"), kit.MenuItem("Quit"))
	file := kit.MenuBarItem("File", fileMenu)
	s := show(t, kit.MenuBar(file, kit.MenuBarItem("Edit", kit.Menu(kit.MenuItem("Undo")))))
	s.click(file)
	require.True(t, opened(fileMenu))

	from := file.Box().Center()
	to := geom.V2(from.X, export.Box().Center().Y)
	if at, closed := glide(s, from, to, func() bool { return opened(fileMenu) }); closed {
		t.Fatalf("the menu closed with the pointer at %v, between its title %v and its first item %v", at, file.Box(),
			export.Box())
	}
	pressHere(s)
	assert.Equal(t, 1, ran, "the item under the pointer runs")
	assert.False(t, opened(fileMenu), "and the menu closes after it")
}

func TestAMenuStaysOpenWhileThePointerCrossesItsItemsDiagonally(t *testing.T) {
	ran := 0
	quit := kit.MenuItem("Quit", ui.OnClick(func() { ran++ }))
	fileMenu := kit.Menu(kit.MenuItem("Export"), kit.SubMenu("Recent", kit.MenuItem("One.tsj"), kit.MenuItem("Two.tsj")),
		kit.MenuItem("Import"), quit)
	file := kit.MenuBarItem("File", fileMenu)
	s := show(t, kit.MenuBar(file, kit.MenuBarItem("Edit", kit.Menu(kit.MenuItem("Undo")))))
	s.click(file)
	require.True(t, opened(fileMenu))

	from := file.Box().Center()
	r := quit.Box()
	to := geom.V2(r.X+r.W-6, r.Y+r.H/2)
	if at, closed := glide(s, from, to, func() bool { return opened(fileMenu) }); closed {
		t.Fatalf("the menu closed with the pointer at %v, crossing its items to %v", at, to)
	}
	pressHere(s)
	assert.Equal(t, 1, ran, "the item at the end of the path runs")
}

// hold presses the left button where the pointer is and keeps it down.
func hold(s *shown) {
	s.in.SetMouseButton(input.MouseLeft, true)
	s.frames(1)
}

// letGo releases the left button where the pointer is.
func letGo(s *shown) {
	s.in.SetMouseButton(input.MouseLeft, false)
	s.frames(2)
}

// menuBarOf is a File title, its menu, its Export item, and how many
// times Export ran.
func menuBarOf() (*ui.Node, *ui.Node, *ui.Node, *int) {
	n := 0
	export := kit.MenuItem("Export", ui.OnClick(func() { n++ }))
	fileMenu := kit.Menu(export, kit.MenuItem("Import"), kit.MenuItem("Quit"))
	file := kit.MenuBarItem("File", fileMenu)
	return file, fileMenu, export, &n
}

func TestAPressOnAMenuTitleOpensItAndAReleaseOnAnItemChoosesIt(t *testing.T) {
	file, fileMenu, export, ran := menuBarOf()
	s := show(t, kit.MenuBar(file, kit.MenuBarItem("Edit", kit.Menu(kit.MenuItem("Undo")))))
	s.in.SetMouse(file.Box().Center())
	s.frames(1)
	hold(s)
	require.True(t, opened(fileMenu), "open on the press, as macOS and Chrome do")
	from := file.Box().Center()
	if at, closed := glide(s, from, geom.V2(from.X, export.Box().Center().Y),
		func() bool { return opened(fileMenu) }); closed {
		t.Fatalf("dragging, the menu closed at %v", at)
	}
	letGo(s)
	assert.Equal(t, 1, *ran, "released on Export, it runs")
	assert.False(t, opened(fileMenu))
}

func TestAPressReleasedOnItsMenuTitleLeavesTheMenuOpenForAClick(t *testing.T) {
	file, fileMenu, export, ran := menuBarOf()
	s := show(t, kit.MenuBar(file))
	s.click(file)
	assert.True(t, opened(fileMenu), "a plain click opens it and leaves it open")
	s.click(export)
	assert.Equal(t, 1, *ran, "then a click on an item runs it")
	assert.False(t, opened(fileMenu))
	s.click(file)
	s.click(file)
	assert.False(t, opened(fileMenu), "a click on the open menu's title still closes it")
}

func TestAPressDraggedOffAMenuAndReleasedOnNothingClosesIt(t *testing.T) {
	file, fileMenu, _, ran := menuBarOf()
	s := show(t, kit.MenuBar(file))
	s.in.SetMouse(file.Box().Center())
	s.frames(1)
	hold(s)
	require.True(t, opened(fileMenu))
	s.in.SetMouse(geom.V2(380, 280))
	s.frames(1)
	letGo(s)
	assert.False(t, opened(fileMenu), "released on nothing, it closes")
	assert.Zero(t, *ran)
}

func TestASelectOpensOnAPressAndChoosesTheOptionReleasedOn(t *testing.T) {
	pick := ui.Select(ui.Option(ui.Text("Box")), ui.Option(ui.Text("Picture")), ui.Option(ui.Text("None")))
	s := show(t, pick)
	s.in.SetMouse(pick.Box().Center())
	s.frames(1)
	hold(s)
	require.True(t, selectOpen(s, pick), "open on the press")
	r := pick.Box()
	s.in.SetMouse(geom.V2(r.X+10, r.Y+r.H+2*r.H+r.H/2))
	s.frames(2)
	letGo(s)
	assert.Equal(t, "None", s.doc.Value(pick), "released on an option, it's chosen")
	assert.False(t, selectOpen(s, pick))

	s.in.SetMouse(pick.Box().Center())
	s.frames(1)
	hold(s)
	letGo(s)
	assert.True(t, selectOpen(s, pick), "a plain click leaves the list open")
	s.click(pick)
	assert.False(t, selectOpen(s, pick))

	s.in.SetMouse(pick.Box().Center())
	s.frames(1)
	hold(s)
	s.in.SetMouse(geom.V2(380, 280))
	s.frames(1)
	letGo(s)
	assert.False(t, selectOpen(s, pick), "released on nothing, the list closes")
	assert.Equal(t, "None", s.doc.Value(pick), "as it was")
}

// selectOpen reports whether the select n shows its list.
func selectOpen(_ *shown, n *ui.Node) bool { return n.State().Has(dom.Open) }

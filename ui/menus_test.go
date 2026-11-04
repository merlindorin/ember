package ui_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func item(text string, args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Attr("role", "menuitem"), ui.Text(text)}, args...)...)
}

func menu(items ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Attr("role", "menu"), ui.Style("position: absolute; top: 100%; left: 0")},
		items...)...)
}

type bar struct {
	l                           *live
	file, edit, view            *ui.Node
	fileMenu, editMenu          *ui.Node
	newItem, open, recent, quit *ui.Node
	recentMenu, one, two        *ui.Node
	undo, redo, cut             *ui.Node
	before                      *ui.Node
}

func menubar(t *testing.T) *bar {
	b := &bar{}
	b.one, b.two = item("One.tsj"), item("Two.tsj")
	b.recentMenu = menu(b.one, b.two)
	b.newItem, b.open, b.quit = item("New"), item("Open"), item("Quit")
	b.recent = ui.Div(ui.Attr("role", "menuitem"), ui.Style("position: relative"), ui.Text("Recent"), b.recentMenu)
	b.fileMenu = menu(b.newItem, b.open, b.recent, b.quit)
	b.undo, b.redo = item("Undo"), item("Redo", ui.Attr("aria-disabled", "true"))
	b.cut = item("Cut")
	b.editMenu = menu(b.undo, b.redo, b.cut)
	b.file = ui.Span(ui.Attr("role", "menuitem"),
		ui.Style("position: relative; display: inline-block; padding: 0 8px"), ui.Text("File"), b.fileMenu)
	b.edit = ui.Span(ui.Attr("role", "menuitem"),
		ui.Style("position: relative; display: inline-block; padding: 0 8px"), ui.Text("Edit"), b.editMenu)
	b.view = ui.Span(ui.Attr("role", "menuitem"), ui.Style("display: inline-block; padding: 0 8px"), ui.Text("View"))
	b.before = ui.Button(ui.Text("Before"))
	b.l = liveDoc(t, b.before, ui.Div(ui.Attr("role", "menubar"), b.file, b.edit, b.view), ui.Button(ui.Text("After")))
	return b
}

func open(n *ui.Node) bool { _, ok := n.Attr("open"); return ok }

func TestAMenubarOpensItsMenusFromTheKeys(t *testing.T) {
	b := menubar(t)
	l := b.l
	require.False(t, open(b.fileMenu))
	require.Zero(t, b.newItem.Box().W, "a closed menu isn't displayed")
	l.doc.Focus(b.before)
	l.frames(1)
	l.key(input.KeyTab)
	assert.True(t, l.doc.Focused(b.file), "the menubar's first item is its Tab stop")

	l.key(input.KeyDown)
	assert.True(t, open(b.fileMenu))
	v, _ := b.file.Attr("aria-expanded")
	assert.Equal(t, "true", v)
	assert.True(t, l.doc.Focused(b.newItem), "Down opens the menu on its first item")
	l.key(input.KeyUp)
	assert.True(t, l.doc.Focused(b.quit), "the arrows wrap in a menu")
	l.key(input.KeyUp)
	l.key(input.KeyRight)
	assert.True(t, open(b.recentMenu), "Right opens a submenu")
	assert.True(t, l.doc.Focused(b.one))
	l.key(input.KeyLeft)
	assert.False(t, open(b.recentMenu), "Left closes it")
	assert.True(t, l.doc.Focused(b.recent))
	l.key(input.KeyRight)
	l.key(input.KeyRight)
	assert.True(t, open(b.editMenu) && !open(b.fileMenu), "Right past a leaf moves to the next menu of the bar")
	assert.True(t, l.doc.Focused(b.undo))
	l.key(input.KeyDown)
	assert.True(t, l.doc.Focused(b.cut), "the disabled Redo is skipped")

	l.key(input.KeyEscape)
	assert.False(t, open(b.editMenu))
	assert.True(t, l.doc.Focused(b.edit), "Escape goes back to the menubar")
	l.key(input.KeyLeft)
	assert.True(t, l.doc.Focused(b.file))

	l.key(input.KeyEnter)
	l.key(input.KeyDown)
	assert.True(t, l.doc.Focused(b.open))
	clicked := len(l.clicks)
	l.key(input.KeyEnter)
	assert.Same(t, b.open, l.clicks[clicked], "Enter chooses an item: Clicked")
	assert.False(t, open(b.fileMenu), "and closes the menus")
	assert.True(t, l.doc.Focused(b.file), "the focus goes back to the menubar")
}

func TestTypingFindsAnItemAndTabClosesTheMenus(t *testing.T) {
	b := menubar(t)
	l := b.l
	l.doc.Focus(b.before)
	l.frames(1)
	l.key(input.KeyTab)
	l.key(input.KeyDown)
	in := l.win.Input()
	in.AddText('q')
	l.frames(2)
	assert.True(t, l.doc.Focused(b.quit), "typeahead: the next item starting with q")
	l.key(input.KeyTab)
	assert.False(t, open(b.fileMenu))
}

func TestThePointerOpensSwitchesAndChooses(t *testing.T) {
	b := menubar(t)
	l := b.l
	l.clickAt(b.file)
	assert.True(t, open(b.fileMenu))
	in := l.win.Input()
	in.SetMouse(b.edit.Box().Center())
	l.frames(2)
	assert.True(t, open(b.editMenu) && !open(b.fileMenu), "hovering another item of the bar switches menus")
	l.clickAt(b.undo)
	assert.Contains(t, l.clicks, b.undo)
	assert.False(t, open(b.editMenu))

	l.clickAt(b.file)
	assert.True(t, open(b.fileMenu))
	l.clickAt(b.file)
	assert.False(t, open(b.fileMenu), "a second click closes it")
	l.clickAt(b.file)
	in.SetMouse(b.view.Box().Center().Add(geomV(0, 200)))
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(2)
	assert.False(t, open(b.fileMenu), "a press outside closes the menus")
}

func TestAutomationSeesMenuItemsByTheirOwnLabels(t *testing.T) {
	b := menubar(t)
	l := b.l
	assert.Equal(t, []string{"File", "Edit", "View"}, l.roles(automation.RoleMenuItem), "closed menus are hidden")
	l.clickAt(b.file)
	assert.Equal(t, []string{"File", "New", "Open", "Recent", "Quit", "Edit", "View"}, l.roles(automation.RoleMenuItem))
}

func geomV(x, y float32) geom.Vec2 { return geom.Vec2{X: x, Y: y} }

func TestAMenubarItemKeepsItsIDWhileItsMenuIsOpen(t *testing.T) {
	b := menubar(t)
	l := b.l
	ids := func() []string {
		_, all := l.auto.Snapshot("", "")
		var out []string
		for _, e := range all {
			if e.Label == "File" {
				out = append(out, e.ID)
			}
		}
		return out
	}
	closed := ids()
	l.clickAt(b.file)
	assert.Equal(t, closed, ids())
}

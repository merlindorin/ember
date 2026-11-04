package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func opened(n *ui.Node) bool { _, ok := n.Attr("open"); return ok }

func TestAMenuBarOpensChoosesAndCloses(t *testing.T) {
	save := kit.MenuItem("Save", kit.Shortcut("Cmd+S"))
	redo := kit.MenuItem("Redo", ui.Attr("aria-disabled", "true"))
	fileMenu := kit.Menu(kit.MenuItem("New", kit.Shortcut("Cmd+N")), save, kit.Separator(),
		kit.SubMenu("Recent", kit.MenuItem("One.tsj")))
	editMenu := kit.Menu(kit.MenuItem("Undo"), redo)
	file, edit := kit.MenuBarItem("File", fileMenu), kit.MenuBarItem("Edit", editMenu)
	s := show(t, kit.MenuBar(file, edit))
	clicks := watch[ui.Clicked](s)
	assert.Zero(t, save.Box().W, "closed menus are hidden")

	s.click(file)
	assert.True(t, opened(fileMenu))
	assert.Greater(t, save.Box().W, float32(0))
	assert.Greater(t, save.Box().Y, file.Box().Y, "under its title")
	assert.Equal(t, automation.RoleMenuItem, s.element(t, "Save").Role, "labelled without its shortcut")

	s.click(save)
	require.Len(t, *clicks, 1)
	assert.Same(t, save, (*clicks)[0].Node)
	assert.False(t, opened(fileMenu))

	s.click(edit)
	s.key(input.KeyDown)
	assert.True(t, s.doc.Focused(editMenu.Children()[0]), "focus on Undo")
	s.key(input.KeyDown)
	assert.True(t, s.doc.Focused(editMenu.Children()[0]), "the disabled Redo is skipped: back to Undo")
	s.key(input.KeyEscape)
	assert.False(t, opened(editMenu))
}

func TestAMenuButtonOpensItsMenu(t *testing.T) {
	m := kit.Menu(kit.MenuItem("Rename"), kit.MenuItem("Delete"))
	b := kit.MenuButton("Actions", m)
	s := show(t, b)
	s.click(b)
	assert.True(t, opened(m))
	assert.Greater(t, m.Box().W, float32(0))
	s.click(b)
	assert.False(t, opened(m))
}

func TestCheckAndRadioMenuItemsTurnAndShortcutsChoose(t *testing.T) {
	grid := kit.CheckMenuItem("Pixel grid", true, kit.Shortcut("Cmd+G"))
	small, large := kit.RadioMenuItem("Small", true), kit.RadioMenuItem("Large", false)
	plain := kit.MenuItem("Preview room…")
	view := kit.Menu(grid, plain, kit.Separator(), small, large)
	bar := kit.MenuBarItem("View", view)
	s := show(t, kit.MenuBar(bar))
	clicks := watch[ui.Clicked](s)
	keys, _ := grid.Attr("aria-keyshortcuts")
	assert.Equal(t, "Cmd+G", keys, "a Shortcut is also the item's aria-keyshortcuts")

	s.click(bar)
	assert.True(t, s.doc.Checked(grid))
	assert.Greater(t, grid.Children()[0].Box().W, float32(0), "a check before its text")
	assert.Equal(t, grid.Children()[1].Box().X, plain.Children()[0].Box().X,
		"a plain item's text lines up with a check item's")
	s.click(grid)
	assert.False(t, s.doc.Checked(grid), "choosing a check item turns it")
	s.click(bar)
	s.click(large)
	assert.True(t, s.doc.Checked(large))
	assert.False(t, s.doc.Checked(small), "a radio item unchecks the others of its menu")
	assert.Len(t, *clicks, 2)
}

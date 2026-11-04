package kit_test

import (
	"slices"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASidebarChoosesAPlaceAcrossItsSections(t *testing.T) {
	home, tilesets := kit.SidebarItem("Home"), kit.SidebarItem("Tilesets")
	walls := kit.SidebarItem("walls.tsj")
	side := kit.Sidebar(kit.SidebarSection("Places", home, tilesets), kit.SidebarSection("Recent", walls))
	s := show(t, side)
	changes := watch[ui.Changed](s)
	assert.Equal(t, "Places", side.Children()[0].Children()[0].Text(),
		"titles keep their text; CSS shows them in capitals")
	assert.Equal(t, automation.RoleSelect, s.element(t, "Tilesets").Role)

	s.click(tilesets)
	require.Len(t, *changes, 1)
	assert.Same(t, side, (*changes)[0].Node)
	assert.Equal(t, 1, s.doc.Selected(side))
	s.key(input.KeyDown)
	assert.Equal(t, 2, s.doc.Selected(side), "down into Recent")
	assert.Greater(t, walls.Box().Y, tilesets.Box().Y+tilesets.Box().H, "under its title")

	props := kit.SidebarItem("props.tsj")
	kit.SetSection(side, "Recent", props, walls)
	s.frames(2)
	assert.True(t, slices.Equal([]*ui.Node{home, tilesets, props, walls}, kit.SidebarItems(side)),
		"a section of that title is replaced")
	assert.True(t, slices.Equal([]*ui.Node{props, walls}, kit.SectionItems(side, "Recent")))
}

package ui_test

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tabbed is a tablist of Video, Audio, Controls and a disabled Game, and
// their panels, Video selected.
func tabbed() (*ui.Node, []*ui.Node, []*ui.Node) {
	var tabs, panels []*ui.Node
	list := ui.Div(ui.Attr("role", "tablist"))
	for i, name := range []string{"Video", "Audio", "Controls", "Game"} {
		selected := "false"
		if i == 0 {
			selected = "true"
		}
		tab := ui.Button(ui.Attr("role", "tab"), ui.Attr("aria-selected", selected),
			ui.Attr("aria-controls", "panel-"+name), ui.Text(name))
		if name == "Game" {
			tab.SetAttr("disabled", "")
		}
		list.Append(tab)
		tabs = append(tabs, tab)
		panels = append(panels, ui.Div(ui.ID("panel-"+name), ui.Attr("role", "tabpanel"), ui.Text(name+" settings")))
	}
	return list, tabs, panels
}

func shownPanels(panels []*ui.Node) []bool {
	var out []bool
	for _, p := range panels {
		out = append(out, p.Box().W > 0)
	}
	return out
}

func TestTabsShowTheSelectedTabsPanel(t *testing.T) {
	list, tabs, panels := tabbed()
	l := liveDoc(t, append([]*ui.Node{list}, panels...)...)
	assert.Equal(t, []bool{true, false, false, false}, shownPanels(panels), "only the selected tab's panel shows")
	assert.Equal(t, 0, l.doc.Selected(list))

	l.clickAt(tabs[1])
	assert.Equal(t, "true", attr(tabs[1], "aria-selected"))
	assert.Equal(t, "false", attr(tabs[0], "aria-selected"))
	assert.Equal(t, []bool{false, true, false, false}, shownPanels(panels))
	assert.True(t, slices.Equal([]*ui.Node{list}, l.changes), "Changed on the tablist")
	assert.Equal(t, 1, l.doc.Selected(list))

	l.clickAt(tabs[1])
	assert.Len(t, l.changes, 1, "the selected tab again changes nothing")

	l.doc.SetSelected(list, 2)
	assert.False(t, has(panels[2], "hidden"), "the panel shows at once, before a frame")
	assert.True(t, has(panels[1], "hidden"))
	l.frames(2)
	assert.Equal(t, []bool{false, false, true, false}, shownPanels(panels))
	assert.Len(t, l.changes, 1, "SetSelected emits nothing")

	assert.Equal(t, []string{"Video", "Audio", "Controls", "Game"}, l.roles(automation.RoleSelect))
}

func TestTabsAreWalkedByTheArrowsAndSelectedAsTheyGo(t *testing.T) {
	before, after := ui.Button(ui.Text("Before")), ui.Button(ui.Text("After"))
	list, tabs, panels := tabbed()
	l := liveDoc(t, append([]*ui.Node{before, list, after}, panels...)...)
	l.doc.Focus(before)
	l.frames(1)

	l.key(input.KeyTab)
	require.True(t, l.doc.Focused(tabs[0]), "Tab reaches the selected tab")
	l.key(input.KeyRight)
	assert.True(t, l.doc.Focused(tabs[1]))
	assert.Equal(t, []bool{false, true, false, false}, shownPanels(panels), "the focused tab is selected")
	l.key(input.KeyRight)
	l.key(input.KeyRight)
	assert.True(t, l.doc.Focused(tabs[0]), "past the disabled Game, back to the first")
	l.key(input.KeyLeft)
	assert.True(t, l.doc.Focused(tabs[2]), "wrapping the other way")
	l.key(input.KeyHome)
	assert.True(t, l.doc.Focused(tabs[0]))
	l.key(input.KeyEnd)
	assert.True(t, l.doc.Focused(tabs[2]), "the last tab that can be chosen")

	l.key(input.KeyTab)
	assert.True(t, l.doc.Focused(after), "one Tab stop for the tablist")
}

func attr(n *ui.Node, name string) string { v, _ := n.Attr(name); return v }

func has(n *ui.Node, name string) bool { _, ok := n.Attr(name); return ok }

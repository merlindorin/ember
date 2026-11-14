package main

import (
	"fmt"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// popUp is a video setting's window mode, and a shape not chosen yet.
func popUp(*gallery) *ui.Node {
	return kit.Form(
		kit.Field("Window mode", kit.PopUp(0, []string{"Windowed", "Borderless", "Fullscreen"})),
		kit.Field("Shape", kit.PopUp(-1, []string{"Rectangle", "Ellipse", "Polygon"}, ui.Placeholder("Choose…"))),
	)
}

// pathPopUp is the folder a tileset tool browses.
func pathPopUp(*gallery) *ui.Node {
	return kit.PathPopUp("/Users/ada/assets/survivors/crypt")
}

// tooltip is a tool bar whose buttons explain themselves on hover or focus.
func tooltip(*gallery) *ui.Node {
	return kit.Row(kit.Gap(12),
		kit.Button("Save", kit.Tooltip(kit.Label("Saves the tileset", kit.Below()))),
		kit.Button("Fill", kit.Tooltip(kit.Label("Fills the area with a tile", kit.Below()))),
		kit.Button("Erase", kit.Tooltip(kit.Label("Clears the tile under the pointer", kit.Below()))),
	)
}

// scroll is a long log of what happened in a run, in a short box.
func scroll(*gallery) *ui.Node {
	args := []ui.Arg{ui.Style("height: 200px; width: 260px")}
	for i := range 30 {
		args = append(args, kit.Label(fmt.Sprintf("Wave %d cleared", i+1), ui.Style("display: block")))
	}
	return kit.Scroll(args...)
}

// form is a player's settings, and a tile's under it lined up with them:
// one label width, the tile's fields one width.
func form(*gallery) *ui.Node {
	player := kit.Form(
		kit.Field("Name", kit.TextInput("Ayla")),
		kit.Field("Volume", kit.Slider(60, kit.Range(0, 100))),
		kit.Separator(kit.Divider()),
		kit.Field("Music", kit.Check("On", ui.Checked(true))),
	)
	tile := kit.Form(
		kit.Field("Width", kit.NumberField(32)),
		kit.Field("Height in tiles", kit.NumberField(2)),
		kit.Field("Seed", kit.TextInput("crypt-7")),
	)
	kit.SetLabelWidth(player, 110)
	kit.SetLabelWidth(tile, 110)
	kit.SetFieldWidth(tile, 120)
	return kit.Column(kit.Gap(18), ui.Style("width: 320px"), player, tile)
}

// tabs is a settings window's pages.
func tabs(*gallery) *ui.Node {
	return kit.Tabs(
		kit.Tab("Video", kit.Label("Resolution, VSync and scale")),
		kit.Tab("Audio", kit.Label("Master, music and effects volumes")),
		kit.Tab("Controls", kit.Label("Key bindings")),
	)
}

// table is a level's files, sortable by name and size.
func table(g *gallery) *ui.Node {
	d := g.doc
	var rows []*ui.Node
	for _, f := range [][4]string{
		{"floor.png", "120 KB", "120", "Picture"},
		{"props.tsj", "9 KB", "9", "Tileset"},
		{"rooms.tmj", "61 KB", "61", "Map"},
		{"walls.tsj", "24 KB", "24", "Tileset"},
	} {
		r := kit.TableRow(f[0], f[1], f[3])
		kit.SortValue(r, 1, f[2])
		rows = append(rows, r)
	}
	t := kit.Table([]*ui.Node{
		kit.TableColumn("Name", kit.Sortable()),
		kit.TableColumn("Size", kit.ColumnWidth(70), kit.ColumnAlign("right"), kit.Sortable()),
		kit.TableColumn("Kind", kit.ColumnWidth(90)),
	}, rows)
	kit.SortedBy(d, t, 0, false)
	return t
}

// sidebar is a level editor's places and recent files.
func sidebar(g *gallery) *ui.Node {
	d := g.doc
	s := kit.Sidebar(
		kit.SidebarSection("Places", kit.SidebarItem("Home"), kit.SidebarItem("Desktop"), kit.SidebarItem("Tilesets")),
		kit.SidebarSection("Recent", kit.SidebarItem("walls.tsj"), kit.SidebarItem("props.tsj")),
	)
	s.SetStyle("width: 180px")
	d.SetSelected(s, 2)
	return s
}

// 25 Sidebar: kit.Sidebar of Places, Tilesets chosen and Desktop hovered, and Recent files,
// beside the same in the file dialog's "places" look, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: flex; gap: 16px; align-items: flex-start; }
.kit-sidebar { width: 180px; }
`

func sidebar(d *ui.Document, args ...ui.Arg) {
	desktop, tilesets := kit.SidebarItem("Desktop"), kit.SidebarItem("Tilesets")
	s := kit.Sidebar(
		kit.SidebarSection("Places", kit.SidebarItem("Home"), desktop, tilesets),
		kit.SidebarSection("Recent", kit.SidebarItem("walls.tsj"), kit.SidebarItem("props.tsj")),
	)
	for _, a := range args {
		a.Apply(s)
	}
	d.Body().Append(s)
	d.SetSelected(s, 2)
	pages.Pin(d, desktop, dom.Hover)
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	sidebar(d)
	sidebar(d, ui.Class("kit-places"))
	pages.Run(d, 420, 250)
}

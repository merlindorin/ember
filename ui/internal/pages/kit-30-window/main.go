// 30 Window: kit.Window with a toolbar, body and footer, a compact one, a minimised one,
// and one whose UnifiedToolbar stands for its title bar, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
html { background-color: var(--window-color); }
.at-1 { inset: auto; left: 16px; top: 16px; margin: 0; width: 300px; }
.at-2 { inset: auto; left: 336px; top: 16px; margin: 0; width: 220px; }
.at-3 { inset: auto; left: 336px; top: 200px; margin: 0; width: 220px; }
.at-4 { inset: auto; left: 16px; top: 280px; margin: 0; width: 540px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	inspector := kit.Window("Inspector", ui.Class("at-1"),
		kit.Toolbar(kit.Button("Save"), kit.Button("Revert")),
		kit.Label("Tile 12 of walls.tsj"), kit.Label("solid, occluder group walls"),
		kit.Footer(kit.Label("2 changes", ui.Class("kit-expand")), kit.Button("Close", ui.Class("kit-primary"))))
	tool := kit.Window("Layers", kit.Compact(), ui.Class("at-2"), kit.Label("ground"), kit.Label("walls"),
		kit.Label("props"))
	folded := kit.Window("Minimised", ui.Class("at-3"), ui.Attr("data-minimised", ""), kit.Label("hidden"))
	files := kit.Window("", kit.UnifiedToolbar(), ui.Class("at-4"),
		kit.Toolbar(kit.Button("‹"), kit.Button("›"),
			kit.Label("Documents", ui.Class("kit-expand")), kit.Button("Search")),
		kit.Label("walls.tsj"), kit.Label("floor.tsj"))
	d.Body().Append(inspector, tool, folded, files)
	for _, w := range []*ui.Node{inspector, tool, folded, files} {
		d.Show(w)
	}
	pages.Run(d, 580, 440)
}

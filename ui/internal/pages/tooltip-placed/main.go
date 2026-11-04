// Tooltips placed by the window: one at its right edge shown on the left,
// one too wide for either side held inside it, both pinned open.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); position: relative; margin: 0; height: 200px; }
.edge { position: absolute; right: 12px; top: 20px; }
.middle { position: absolute; left: 150px; top: 110px; }
.wide { white-space: normal; width: 260px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	edge := kit.Label("Save", ui.Class("edge"), kit.Tooltip(kit.Label("Saves the file")))
	middle := kit.Label("Grid", ui.Class("middle"), kit.Tooltip(kit.Label("Shows the grid over the map, one line per tile",
		ui.Class("wide"))))
	d.Body().Append(edge, middle)
	pages.Pin(d, edge, dom.Hover)
	pages.Pin(d, middle, dom.Hover)
	pages.Run(d, 320, 200)
}

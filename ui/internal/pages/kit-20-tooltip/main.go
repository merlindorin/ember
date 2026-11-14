// 20 Tooltip: kit.Tooltip's tips pinned open (hovered, focused from the keys, on the left,
// below, on a button and as a panel of labels), on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 1fr;
  gap: 30px 16px; align-items: baseline; }
.slot { padding-left: 110px; }
.stack { display: flex; flex-direction: column; gap: 2px; }
.below-row { margin-bottom: 26px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, n *ui.Node, s dom.State) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), ui.Div(ui.Class("slot"), n))
		if s != 0 {
			pages.Pin(d, n, s)
		}
	}
	row("closed", kit.Label("Save", kit.Tooltip(kit.Label("Saves the tileset"))), 0)
	row("hovered", kit.Label("Save", kit.Tooltip(kit.Label("Saves the tileset"))), dom.Hover)
	row("keyboard focus", kit.Label("Open", kit.Tooltip(kit.Label("Opens a tileset"))), dom.Focus|dom.FocusVisible)
	row("left", kit.Label("Undo", kit.Tooltip(kit.Label("Undoes", ui.Class("kit-left")))), dom.Hover)
	row("below", kit.Label("Grid", ui.Class("below-row"), kit.Tooltip(kit.Label("Shows the grid", ui.Class("kit-below")))),
		dom.Hover)
	row("button", kit.Button("Delete", kit.Tooltip(kit.Label("Deletes the tile"))), dom.Hover)
	row("panel", kit.Label("solid", kit.Tooltip(ui.Div(ui.Class("kit-below"), ui.Div(ui.Class("stack"),
		kit.Label("solid", ui.Class("kit-alert-title")),
		kit.Label("Blocks walkers and light.", ui.Class("kit-alert-message")))))), dom.Hover)
	pages.Run(d, 440, 470)
}

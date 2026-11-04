// 31 Alert: a kit.Alert asking to save, a destructive button on the left, the primary
// one focused, over a dimmed page on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `body { background-color: var(--window-color); padding: 16px; color: var(--light-text); }
.dim { position: fixed; inset: 0; background-color: rgba(0, 0, 0, 0.4); }`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	alert := kit.Alert("Save the tileset?", "Your changes to walls.tsj will be lost if you close it without saving.",
		kit.Button("Don't save", ui.Class("kit-destructive")),
		kit.Button("Cancel"),
		kit.Button("Save", ui.Class("kit-primary")),
	)
	d.Body().Append(
		ui.P(ui.Text("walls.tsj — 48 tiles, 3 edited")),
		ui.Div(ui.Class("dim")),
		alert,
	)
	d.Show(alert)
	pages.Run(d, 560, 300)
}

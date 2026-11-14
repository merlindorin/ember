// 05 Separator: kit.Separator in a list, a divider in a dialog, and vertical separators and dividers each in a toolbar,
// on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 220px;
  gap: 12px 16px; align-items: baseline; }
.list, .dialog { background-color: var(--panel-color); border: 1px solid var(--border-color); border-radius: 4px;
  padding: 4px 0; }
.list .kit-label { display: block; padding: 2px 10px; }
.dialog { padding: 10px; }
.dialog .kit-separator { margin: 8px 0; }
.kit-toolbar { display: flex; align-items: center; gap: 0; height: 28px; background-color: var(--window-bar);
  padding: 0 6px; }
.kit-toolbar .kit-label { padding: 0 6px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, n *ui.Node) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), n)
	}
	row("list", ui.Div(ui.Class("list"),
		kit.Label("New file"), kit.Label("Open…"), kit.Separator(),
		kit.Label("Save"), kit.Label("Save as…"), kit.Separator(), kit.Label("Quit")))
	row("divider", ui.Div(ui.Class("dialog"),
		kit.Label("Delete this file?", ui.Class("kit-alert-title")),
		kit.Separator(ui.Class("kit-divider")),
		kit.Label("This can't be undone.", ui.Class("kit-alert-message"))))
	row("vertical", ui.Div(ui.Class("kit-toolbar"),
		kit.Label("Cut"), kit.Label("Copy"), kit.Separator(ui.Class("kit-vertical")),
		kit.Label("Paste"), kit.Separator(ui.Class("kit-vertical")), kit.Label("Undo")))
	row("vertical divider", ui.Div(ui.Class("kit-toolbar"),
		kit.Label("Cut"), kit.Label("Copy"), kit.Separator(ui.Class("kit-vertical", "kit-divider")),
		kit.Label("Paste"), kit.Separator(ui.Class("kit-vertical", "kit-divider")), kit.Label("Undo")))
	pages.Run(d, 360, 370)
}

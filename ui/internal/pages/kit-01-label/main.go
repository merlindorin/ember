// 01 Label: every variant of kit.Label on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 200px;
  gap: 10px 16px; align-items: baseline; }
.box { border: 1px dashed #5a524a; }
`

const long = "The quick brown fox jumps over the lazy dog"

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, label *ui.Node) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), ui.Div(ui.Class("box"), label))
	}
	row("default", kit.Label("Health: 100"))
	row("wrap", kit.Label(long, ui.Class("kit-wrap")))
	row("truncate", kit.Label(long, ui.Class("kit-truncate")))
	row("center", kit.Label("centred", ui.Class("kit-center")))
	row("right", kit.Label("right", ui.Class("kit-right")))
	row("caps", kit.Label("NAME", ui.Class("kit-caps")))
	row("form-label", kit.Label("light radius", ui.Class("kit-form-label")))
	row("alert-title", kit.Label("Delete the tileset?", ui.Class("kit-alert-title")))
	row("alert-message", kit.Label("This cannot be undone.", ui.Class("kit-alert-message")))
	pages.Run(d, 340, 360)
}

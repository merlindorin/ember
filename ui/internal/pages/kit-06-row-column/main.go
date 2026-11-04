// 06 Row/Column: kit.Row and kit.Column with every placement, gap and padding, on
// ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 240px;
  gap: 8px 16px; align-items: center; }
.frame { border: 1px dashed #5a524a; }
.chip { background-color: var(--button-color); border: 1px solid var(--border-color); border-radius: 3px;
  min-width: 18px; min-height: 18px; }
.tall { height: 32px; }
.wide { width: 40px; }
`

func chip(classes ...string) *ui.Node {
	return ui.Div(ui.Class(append([]string{"chip"}, classes...)...))
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, n *ui.Node) {
		n.AddClass("frame")
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), n)
	}
	row("row", kit.Row(chip(), chip("wide"), chip()))
	row("column", kit.Column(chip("wide"), chip(), chip("wide")))
	row("gap 12", kit.Row(kit.Gap(12), chip(), chip(), chip()))
	row("padding 8", kit.Row(kit.Padding(8), chip(), chip()))
	row("expand", kit.Row(chip(), chip("kit-expand"), chip()))
	row("justify-center", kit.Row(ui.Class("kit-justify-center"), chip(), chip()))
	row("justify-end", kit.Row(ui.Class("kit-justify-end"), chip(), chip()))
	row("justify-between", kit.Row(ui.Class("kit-justify-between"), chip(), chip(), chip()))
	row("stretch", kit.Row(chip("tall"), chip(), chip()))
	row("align-start", kit.Row(ui.Class("kit-align-start"), chip("tall"), chip(), chip()))
	row("align-center", kit.Row(ui.Class("kit-align-center"), chip("tall"), chip(), chip()))
	row("align-end", kit.Row(ui.Class("kit-align-end"), chip("tall"), chip(), chip()))
	row("toolbar", kit.Row(ui.Class("kit-toolbar"), chip(), chip()))
	row("body", kit.Column(ui.Class("kit-body"), chip("wide"), chip("wide")))
	row("footer", kit.Row(ui.Class("kit-footer", "kit-justify-end"), chip("wide"), chip("wide")))
	pages.Run(d, 420, 680)
}

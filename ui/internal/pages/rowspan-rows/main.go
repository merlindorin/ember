// Row backgrounds under cells that span rows, in high contrast: each row
// paints only behind its own cells, a spanning cell keeping its first
// row's colour over its whole box, as Chrome paints rows; with
// border-spacing, the gaps show the table's background.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; }
table { border-spacing: 0; margin-bottom: 12px; background-color: #3060c0; }
table.gaps { border-spacing: 6px; }
td { width: 60px; height: 24px; padding: 0; }
.black { background-color: #000; }
.white { background-color: #fff; }
.red { background-color: #e02020; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"rows.css": {Data: []byte(sheet)}}, "rows.css"))
	table := func(class string) *ui.Node {
		return ui.Table(ui.Class(class), ui.Tbody(
			ui.Tr(ui.Class("black"), ui.Td(ui.Attr("rowspan", "2")), ui.Td(), ui.Td()),
			ui.Tr(ui.Class("white"), ui.Td(), ui.Td()),
			ui.Tr(ui.Class("red"), ui.Td(), ui.Td(ui.Attr("rowspan", "2")), ui.Td()),
			ui.Tr(ui.Class("black"), ui.Td(), ui.Td()),
		))
	}
	d.Body().Append(ui.Div(ui.Style("display: flex; gap: 16px; align-items: flex-start"), table(""), table("gaps")))
	pages.Run(d, 460, 160)
}

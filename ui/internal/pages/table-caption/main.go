// Tables styled as a game styles them: a border, padding and a background
// round the grid, the caption outside them above or below, as Chrome's
// table wrapper puts it, a round one outlined round its caption too; cells
// spanning columns and rows.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; display: flex; gap: 16px; align-items: flex-start; }
table { border: 2px solid #345; padding: 6px; background-color: #e4ecf7; border-spacing: 3px; }
td { padding: 2px 6px; background-color: #fff; }
caption { padding: 4px; background-color: #fde9c8; }
.below { caption-side: bottom; }
.round { border-radius: 10px; outline: 2px solid #c33; outline-offset: 2px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"table-caption.css": {Data: []byte(sheet)}}, "table-caption.css"))
	cell := func(text string, attrs ...string) *ui.Node {
		args := []ui.Arg{ui.Text(text)}
		for i := 0; i+1 < len(attrs); i += 2 {
			args = append(args, ui.Attr(attrs[i], attrs[i+1]))
		}
		return ui.Td(args...)
	}
	d.Body().Append(
		ui.Table(ui.Caption(ui.Text("Scores")),
			ui.Tbody(ui.Tr(cell("Ada"), cell("12")), ui.Tr(cell("Bo"), cell("7")))),
		ui.Table(ui.Class("below", "round"), ui.Caption(ui.Text("Under the grid")),
			ui.Tbody(ui.Tr(cell("Cy"), cell("31")))),
		ui.Table(ui.Caption(ui.Text("Spans")),
			ui.Tbody(
				ui.Tr(cell("across two", "colspan", "2"), cell("x")),
				ui.Tr(cell("down", "rowspan", "2"), cell("a"), cell("b")),
				ui.Tr(cell("c"), cell("d")))),
	)
	pages.Run(d, 420, 160)
}

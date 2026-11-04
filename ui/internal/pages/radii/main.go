// Corners with their own radii: one corner past half the box, radii that
// overlap and scale down, and borders thicker than a corner's radius.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; }
.row { display: flex; gap: 16px; margin-bottom: 16px; align-items: flex-start; }
.fill { background-color: #3a6fd8; }
.ring { border: 8px solid #c0392b; background-color: #f4e9d8; }
.thin { border: 1px solid #222; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"radii.css": {Data: []byte(sheet)}}, "radii.css"))
	box := func(class, style string) *ui.Node { return ui.Div(ui.Class(class), ui.Style(style)) }
	d.Body().Append(
		ui.Div(ui.Class("row"),
			box("fill", "width: 60px; height: 60px; border-radius: 40px 0 0 0"),
			box("fill", "width: 80px; height: 60px; border-radius: 50px 50px 10px 10px"),
			box("fill", "width: 100px; height: 48px; border-radius: 30px 6px"),
			box("fill", "width: 90px; height: 50px; border-radius: 0 24px 0 24px"),
		),
		ui.Div(ui.Class("row"),
			box("ring", "width: 90px; height: 60px; border-radius: 4px 20px 4px 20px"),
			box("ring", "width: 80px; height: 60px; border-radius: 40px 0 12px 0"),
			box("thin", "width: 90px; height: 50px; border-radius: 16px 2px 16px 2px"),
			box("thin", "width: 70px; height: 50px; border-radius: 0 0 25px 25px"),
		),
	)
	pages.Run(d, 520, 190)
}

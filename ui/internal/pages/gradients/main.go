// Linear gradients: angles, sides and corners, three stops and more,
// positioned and hard stops, and gradients under rounded borders.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; }
.row { display: flex; gap: 12px; margin-bottom: 12px; }
.g { width: 80px; height: 56px; }
.ring { border: 6px solid #2c2c2c; border-radius: 18px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"gradients.css": {Data: []byte(sheet)}}, "gradients.css"))
	g := func(class, bg string) *ui.Node {
		return ui.Div(ui.Class("g", class), ui.Style("background-image: "+bg))
	}
	d.Body().Append(
		ui.Div(ui.Class("row"),
			g("", "linear-gradient(#e63946, #1d3557)"),
			g("", "linear-gradient(to right, #000, #fff)"),
			g("", "linear-gradient(45deg, #f1c40f, #8e44ad)"),
			g("", "linear-gradient(0.6turn, #16a085, #f39c12)"),
			g("", "linear-gradient(to bottom left, #2980b9, #ecf0f1)"),
		),
		ui.Div(ui.Class("row"),
			g("", "linear-gradient(to right, red, yellow, lime, cyan, blue)"),
			g("", "linear-gradient(to right, #222 20%, #ddd 30%, #c0392b 70%, #fff)"),
			g("", "linear-gradient(to bottom, #3498db 50%, #e67e22 50%)"),
			g("", "linear-gradient(to right, #8e44ad 20px, #27ae60 20px 50px, #f1c40f 50px)"),
			g("", "linear-gradient(120deg, rgba(0,0,0,0) 10%, rgba(41,128,185,0.8) 90%)"),
		),
		ui.Div(ui.Class("row"),
			g("ring", "linear-gradient(#ff7e5f, #feb47b)"),
			g("ring", "linear-gradient(to right, #43cea2, #185a9d)"),
			ui.Div(ui.Class("g"), ui.Style("border-radius: 22px; background-image: linear-gradient(135deg, #667eea, #764ba2)")),
			ui.Div(ui.Class("g"),
				ui.Style("border-radius: 10px; background: linear-gradient(to top, #000 0%, #555 40%, #eee 100%)")),
		),
	)
	pages.Run(d, 520, 240)
}

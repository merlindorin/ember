// Inset box-shadows: hard frames, offsets, spreads and blurs, on light
// and dark boxes, square and rounded, then blurred outer shadows and
// glows, under opaque and translucent boxes, against Chrome's.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; }
.row { display: flex; gap: 16px; margin-bottom: 16px; }
.box { width: 80px; height: 44px; background-color: #f4f1ec; }
.dark .box { background-color: #2b2722; }
.dark { background-color: #1b1714; padding: 8px; }
.panel { width: 80px; height: 44px; border: 1px solid #3a332c; border-radius: 8px; background-color: #2b2722; }
.field { width: 80px; height: 28px; border: 1px solid #b9b2a6; border-radius: 12px; background-color: #fbf8f3; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"insets.css": {Data: []byte(sheet)}}, "insets.css"))
	shadows := []string{
		"inset 0 0 0 2px #3a6fd8",
		"inset 3px 2px 0 rgba(0,0,0,0.35)",
		"inset 0 2px 6px rgba(0,0,0,0.4)",
		"inset 0 0 12px 2px rgba(0,0,0,0.3)",
	}
	for _, theme := range []string{"", "dark"} {
		row := ui.Div(ui.Class("row", theme))
		for _, s := range shadows {
			if theme == "dark" {
				s += ", inset 0 1px 0 rgba(255,255,255,0.08)"
			}
			row.Append(ui.Div(ui.Class("box"), ui.Style("box-shadow: "+s)))
		}
		d.Body().Append(row)
	}
	rounded := ui.Div(ui.Class("row", "dark"),
		ui.Div(ui.Class("panel"),
			ui.Style("box-shadow: inset 0 1px 0 rgba(255,255,255,0.08), inset 0 0 12px rgba(0,0,0,0.5)")),
		ui.Div(ui.Class("panel"), ui.Style("box-shadow: inset 0 0 0 2px #c79a4a")),
		ui.Div(ui.Class("panel"), ui.Style("border-radius: 16px 4px; box-shadow: inset 4px 4px 8px rgba(0,0,0,0.6)")),
	)
	light := ui.Div(ui.Class("row"),
		ui.Div(ui.Class("field"), ui.Style("box-shadow: inset 0 1px 2px rgba(0,0,0,0.25)")),
		ui.Div(ui.Class("field"), ui.Style("box-shadow: inset 0 0 0 3px rgba(58,111,216,0.5)")),
		ui.Div(ui.Class("field"), ui.Style("box-shadow: inset 0 2px 6px 1px rgba(0,0,0,0.3)")),
	)
	outer := ui.Div(ui.Class("row", "dark"), ui.Style("padding: 20px 16px"),
		ui.Div(ui.Class("box"), ui.Style("box-shadow: 0 4px 10px rgba(0,0,0,0.8)")),
		ui.Div(ui.Class("panel"), ui.Style("box-shadow: 0 0 18px #e0a040")),
		ui.Div(ui.Class("panel"), ui.Style("border-radius: 16px 4px; box-shadow: 3px 3px 8px 2px rgba(0,0,0,0.9)")),
		ui.Div(ui.Class("box"), ui.Style("border-radius: 22px; box-shadow: 0 0 12px 4px rgba(224,160,64,0.6)")),
	)
	translucent := ui.Div(ui.Class("row", "dark"), ui.Style("padding: 20px 16px"),
		ui.Div(ui.Class("box"), ui.Style("background-color: rgba(43,39,34,0.5); box-shadow: 0 0 14px 2px #e0a040")),
		ui.Div(ui.Class("panel"),
			ui.Style("background-color: rgba(43,39,34,0.5); box-shadow: 4px 6px 10px rgba(255,255,255,0.5)")),
		ui.Div(ui.Class("box"),
			ui.Style("background-color: transparent; border: 1px solid #e0a040; box-shadow: 6px 4px 0 #3a6fd8")),
	)
	d.Body().Append(rounded, light, outer, translucent)
	pages.Run(d, 420, 500)
}

// Canvas elements drawn by the game and sized as Chrome sizes them: the default 300 by 150, the
// width and height attributes, CSS keeping their ratio, and one stretched
// in a flex row, one in block flow, each with a background and border of its own.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; }
body { padding: 12px; margin: 0; display: flex; flex-wrap: wrap; align-items: flex-start; gap: 8px; }
canvas { background-color: #2b2722; border: 2px solid #ad9773; padding: 4px; }
.half { width: 60px; }
.flow { width: 100%; }
.flow canvas { display: block; }
.row { display: flex; gap: 8px; height: 70px; width: 100%; }
.row canvas { flex: 1; min-width: 0; }
`

// blank draws a room plan in whatever size it is given.
func blank(c *twod.Canvas) {
	s := c.Size()
	c.Rect(geom.Rect{X: 4, Y: 4, W: s.X / 2, H: s.Y / 2}, color.Hex("#6b8f3a"))
	c.Circle(geom.V2(s.X*0.7, s.Y*0.6), min(s.X, s.Y)/4, color.Hex("#c8553d"))
	c.Line(geom.V2(0, s.Y-2), geom.V2(s.X, 2), 2, color.Hex("#f4ede4"))
	c.Text(geom.V2(6, s.Y-6), "N", twod.FontSize(12), twod.Color(color.Hex("#f4ede4")), twod.Baseline())
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(
		ui.Canvas(blank),
		ui.Canvas(blank, ui.Attr("width", "120"), ui.Attr("height", "60")),
		ui.Canvas(blank, ui.Class("half"), ui.Attr("width", "120"), ui.Attr("height", "60")),
		ui.Div(ui.Class("flow"), ui.Canvas(blank, ui.Class("half"), ui.Attr("width", "120"), ui.Attr("height", "60"))),
		ui.Div(ui.Class("row"),
			ui.Canvas(blank, ui.Attr("width", "40"), ui.Attr("height", "20")),
			ui.Canvas(blank, ui.Attr("width", "40"), ui.Attr("height", "20"))),
	)
	pages.Run(d, 520, 300)
}

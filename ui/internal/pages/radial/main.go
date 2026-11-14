// Radial gradients: the pixel HUD's two, each size keyword for a circle
// and an ellipse, a centre off the middle, and rings of hard stops.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; }
.row { display: flex; gap: 12px; margin-bottom: 12px; }
.g { width: 88px; height: 56px; background-color: #3a6fd8; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"radial.css": {Data: []byte(sheet)}}, "radial.css"))
	g := func(bg string) *ui.Node { return ui.Div(ui.Class("g"), ui.Style("background-image: "+bg)) }
	d.Body().Append(
		ui.Div(ui.Class("row"),
			g("radial-gradient(#3a2a1e 40%, #120c09)"),
			g("radial-gradient(circle at 50% 50%, transparent 30%, rgba(0,0,0,.55) 85%)"),
			g("radial-gradient(circle closest-side, #f1c40f, #c0392b)"),
			g("radial-gradient(closest-side, #f1c40f, #c0392b)"),
		),
		ui.Div(ui.Class("row"),
			g("radial-gradient(circle farthest-side at 30% 40%, #fff, #000)"),
			g("radial-gradient(ellipse closest-corner at 20px 20px, #2a9d8f, #264653)"),
			g("radial-gradient(circle, #e63946 0 20%, #f1faee 20% 40%, #a8dadc 40% 60%, #457b9d 60%)"),
			ui.Div(ui.Class("g"),
				ui.Style("border-radius: 16px; background-image: radial-gradient(circle at 70% 30%, #fff 0, #e9c46a 30%, "+
					"#9b5f2b 100%)")),
		),
	)
	pages.Run(d, 440, 160)
}

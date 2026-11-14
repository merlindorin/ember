// Conic gradients: cooldown sweeps of hard stops at several fills, in
// rounded slots as the HUD draws them, a colour wheel, and a start angle
// and a centre off the middle.
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
.g { width: 64px; height: 64px; }
.slot { width: 40px; height: 40px; border-radius: 6px; background-color: #3a6fd8; }
.sweep { width: 100%; height: 100%; border-radius: 6px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"conic.css": {Data: []byte(sheet)}}, "conic.css"))
	sweep := func(done string) *ui.Node {
		return ui.Div(ui.Class("slot"), ui.Div(ui.Class("sweep"),
			ui.Style("background-image: conic-gradient(transparent 0 "+done+", rgba(0,0,0,.72) "+done+" 100%)")))
	}
	g := func(bg string) *ui.Node { return ui.Div(ui.Class("g"), ui.Style("background-image: "+bg)) }
	d.Body().Append(
		ui.Div(ui.Class("row"), sweep("10%"), sweep("35%"), sweep("50%"), sweep("70%"), sweep("95%")),
		ui.Div(ui.Class("row"),
			g("conic-gradient(red, yellow, lime, cyan, blue, magenta, red)"),
			g("conic-gradient(from 90deg, #222, #eee)"),
			g("conic-gradient(from 45deg at 25% 75%, #e63946 0 90deg, #1d3557 90deg 180deg, #f1c40f 180deg 270deg, #2a9d8f "+
				"270deg)"),
			ui.Div(ui.Class("g"),
				ui.Style("border-radius: 50%; background-image: conic-gradient(#8f6cf0 0 62%, #2a2230 62% 100%)")),
		),
	)
	pages.Run(d, 360, 180)
}

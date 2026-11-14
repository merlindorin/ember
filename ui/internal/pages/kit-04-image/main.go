// 04 Image: kit.ColorRect, kit.NinePatch, kit.Image in each of the old ember/ui's stretch modes and
// kit.TileImage, on ember's window colour; the picture is the 24×24 panel of example 22.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

//go:embed panel.png
var panel []byte

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 220px;
  gap: 10px 16px; align-items: center; }
.text { align-self: baseline; }
.box { border: 1px dashed #5a524a; }
.swatch { height: 24px; }
.half { width: 50%; }
.dialog { width: 200px; padding: 0; }
.kit-button { width: 90px; text-align: center; }
.pillar { width: 16px; height: 80px; }
.bar { width: 200px; height: 16px; }
.frame { display: block; width: 96px; height: 40px; }
.tiles { width: 120px; height: 48px; }
`

func main() {
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}, "panel.png": {Data: panel}}
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(files, "page.css"))
	row := func(name string, n *ui.Node, classes ...string) {
		d.Body().Append(kit.Label(name, ui.Class(append([]string{"kit-form-label"}, classes...)...)),
			ui.Div(ui.Class(append([]string{"box"}, classes...)...), n))
	}
	row("color rect", kit.ColorRect("var(--accent-color)", ui.Class("swatch")))
	row("translucent", kit.ColorRect("rgba(255, 255, 255, 0.25)", ui.Class("swatch", "half")))
	row("nine-patch", kit.NinePatch("panel.png", 8, 8, 8, 8, ui.Class("dialog"),
		kit.Label("Delete this file?", ui.Class("kit-alert-title")),
		kit.Label("This can't be undone.", ui.Class("kit-alert-message", "kit-wrap"))), "text")
	row("nine-patch button", kit.NinePatch("panel.png", 8, 8, 8, 8, ui.Class("kit-button"), kit.Label("Save")), "text")
	row("nine-patch bar", kit.NinePatch("panel.png", 8, 8, 8, 8, ui.Class("bar")))
	row("image", kit.Image("panel.png"))
	row("scale", kit.Image("panel.png", ui.Class("frame")))
	row("keep", kit.Image("panel.png", ui.Class("frame", "kit-keep")))
	row("keep-centered", kit.Image("panel.png", ui.Class("frame", "kit-keep-centered")))
	row("keep-aspect", kit.Image("panel.png", ui.Class("frame", "kit-keep-aspect")))
	row("keep-aspect-centered", kit.Image("panel.png", ui.Class("frame", "kit-keep-aspect-centered")))
	row("keep-aspect-covered", kit.Image("panel.png", ui.Class("frame", "kit-keep-aspect-covered")))
	row("tile", kit.TileImage("panel.png", ui.Class("tiles")))
	pages.Run(d, 400, 720)
}

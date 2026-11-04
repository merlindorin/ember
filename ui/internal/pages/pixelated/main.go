// image-rendering: pixelated: imgs and a border-image's middle scaled up
// from a 24×24 picture, against Chrome's.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

//go:embed panel.png
var panel []byte

const sheet = `
html { background-color: #1b1714; }
body { padding: 12px; display: grid; grid-template-columns: repeat(3, 120px); gap: 12px; }
.pixelated { image-rendering: pixelated; }
img { display: block; width: 96px; height: 96px; }
img.wide { width: 120px; height: 60px; }
.nine { width: 120px; height: 96px; border: 8px solid transparent; border-image: url(panel.png) 8 fill;
  box-sizing: border-box; }
`

func main() {
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}, "panel.png": {Data: panel}}
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"))
	d.Body().Append(
		ui.Img(ui.Attr("src", "panel.png"), ui.Class("pixelated")),
		ui.Img(ui.Attr("src", "panel.png"), ui.Class("pixelated", "wide")),
		ui.Div(ui.Class("nine", "pixelated")),
	)
	pages.Run(d, 420, 130)
}

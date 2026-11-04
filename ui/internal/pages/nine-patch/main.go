// One nine-patch at a few sizes, its slices stretched: where each slice's
// edges land and how its strips fill, against Chrome's border-image.
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
body { padding: 12px; margin: 0; display: flex; flex-wrap: wrap; align-items: flex-start; gap: 12px; }
.patch { border-style: solid; border-color: transparent; border-width: 8px; border-image: url("panel.png") 8 fill; }
.a { width: 40px; height: 40px; }
.b { width: 200px; height: 60px; }
.c { width: 120px; height: 16px; }
.d { width: 23px; height: 101px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "panel.png": {Data: panel}},
		"page.css"))
	for _, size := range []string{"a", "b", "c", "d"} {
		d.Body().Append(ui.Div(ui.Class("patch", size)))
	}
	pages.Run(d, 360, 160)
}

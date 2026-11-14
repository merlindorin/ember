// Rounded corners at radii 2, 4, 8 and 12: filled, with a 1px border and
// with a 2px border, and a dark theme's panels, the corners' edge pixels
// against Chrome's.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; }
.row { display: flex; gap: 16px; margin-bottom: 16px; }
.box { width: 64px; height: 40px; }
.fill { background-color: #3a6fd8; }
.b1 { border: 1px solid #222; }
.b2 { border: 2px solid #c0392b; background-color: #f4e9d8; }
.dark { background-color: #1b1714; padding: 8px; }
.panel { border: 1px solid #3a332c; background-color: #2b2722; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"corners.css": {Data: []byte(sheet)}}, "corners.css"))
	for _, kind := range []string{"fill", "b1", "b2"} {
		row := ui.Div(ui.Class("row"))
		for _, r := range []int{2, 4, 8, 12} {
			row.Append(ui.Div(ui.Class("box", kind), ui.Style(fmt.Sprintf("border-radius: %dpx", r))))
		}
		d.Body().Append(row)
	}
	dark := ui.Div(ui.Class("row", "dark"))
	for _, r := range []int{2, 4, 8, 12} {
		dark.Append(ui.Div(ui.Class("box", "panel"), ui.Style(fmt.Sprintf("border-radius: %dpx", r))))
	}
	d.Body().Append(dark)
	pages.Run(d, 360, 270)
}

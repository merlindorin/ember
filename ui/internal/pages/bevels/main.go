// Inset and outset borders as Chrome draws them: an hr, and boxes with 1
// and 4 pixel sides in grey, red and black, the dark shade on the top and
// left of an inset and the bottom and right of an outset, mitered corners.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 12px; margin-top: 12px; }
.box { width: 48px; height: 32px; background-color: #ffffff; }
.in { border: 4px inset gray; }
.out { border: 4px outset gray; }
.red-in { border: 4px inset #e03c31; }
.red-out { border: 4px outset #e03c31; }
.black-in { border: 4px inset black; }
.thin-in { border: 1px inset gray; }
.thin-out { border: 1px outset gray; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := ui.Div(ui.Class("row"))
	for _, c := range []string{"in", "out", "red-in", "red-out", "black-in", "thin-in", "thin-out"} {
		row.Append(ui.Div(ui.Class("box", c)))
	}
	d.Body().Append(ui.Hr(), row)
	pages.Run(d, 480, 120)
}

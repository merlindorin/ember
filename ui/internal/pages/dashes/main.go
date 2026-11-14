// Dashed and dotted borders, 1 to 4px wide, on boxes of a few lengths:
// where the dashes and dots fall against Chrome's.
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
.row { display: flex; gap: 14px; margin-bottom: 14px; align-items: flex-start; }
.box { border-color: #222; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"dashes.css": {Data: []byte(sheet)}}, "dashes.css"))
	for _, kind := range []string{"dashed", "dotted"} {
		for _, w := range []int{1, 2, 3, 4} {
			row := ui.Div(ui.Class("row"))
			for _, size := range [][2]int{{40, 24}, {73, 31}, {120, 40}} {
				row.Append(ui.Div(ui.Class("box"), ui.Style(fmt.Sprintf("border: %dpx %s #222; width: %dpx; height: %dpx", w, kind,
					size[0], size[1]))))
			}
			d.Body().Append(row)
		}
	}
	pages.Run(d, 300, 520)
}

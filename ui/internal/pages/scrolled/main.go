// A list of rows scrolled inside a scroller so rows are cut at both of its
// edges, and the page below the fold: what culling keeps against Chrome's,
// every row that meets the scrollport drawn.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; color: #f4ede4; }
body { padding: 12px; }
.scroller { width: 260px; height: 190px; overflow: auto; border: 1px solid #ad9773; }
.row { height: 24px; padding: 0 8px; line-height: 24px; border-bottom: 1px solid #3a332c; }
.row:nth-child(odd) { background-color: #2b2722; }
.below { margin-top: 12px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	sc := ui.Div(ui.Class("scroller"))
	for i := range 200 {
		sc.Append(ui.Div(ui.Class("row"), ui.Text(fmt.Sprintf("row %d · walls · solid", i))))
	}
	d.Body().Append(sc)
	for i := range 40 {
		d.Body().Append(ui.Div(ui.Class("row", "below"), ui.Text(fmt.Sprintf("below %d", i))))
	}
	dom.Data(sc).Scroll = geom.V2(0, 1337)
	pages.Run(d, 300, 320)
}

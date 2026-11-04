// A closed details whose body an author rule makes a flex row, which
// Chrome doesn't show, beside an open one that shows it.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
body { margin: 0; padding: 12px; background-color: #f4f4f4; }
.body { display: flex; gap: 6px; padding: 6px; background-color: #3a6ea5; color: #fff; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	body := func(text string) *ui.Node {
		return ui.Div(ui.Class("body"), ui.Span(ui.Text(text)), ui.Span(ui.Text("more")))
	}
	d.Body().Append(
		ui.Details(ui.Summary(ui.Text("closed")), body("hidden")),
		ui.Details(ui.Open(), ui.Summary(ui.Text("open")), body("shown")),
	)
	pages.Run(d, 240, 120)
}

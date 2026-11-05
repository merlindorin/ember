// Edit fields drawn through text-transform: an uppercase input holding
// ß, capitalize and lowercase ones, and a capitalized textarea; their
// values stay as typed.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; }
.row { display: flex; gap: 12px; margin-bottom: 12px; align-items: flex-start; }
input { width: 160px; }
/* An explicit width: sized by cols, the textarea is a point narrower in the CI runners' Chrome. */
textarea { width: 214px; }
.up { text-transform: uppercase; }
.cap { text-transform: capitalize; }
.low { text-transform: lowercase; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"field-case.css": {Data: []byte(sheet)}}, "field-case.css"))
	field := func(class, value string) *ui.Node {
		return ui.Input(ui.Type("text"), ui.Class(class), ui.Value(value))
	}
	d.Body().Append(
		ui.Div(ui.Class("row"),
			field("up", "Straße & café"),
			field("cap", "hello wide world"),
		),
		ui.Div(ui.Class("row"),
			field("low", "MIXED Case Text"),
			field("up", "rune ﬁx and ŝ"),
		),
		ui.Div(ui.Class("row"),
			ui.Textarea(ui.Class("cap"), ui.Attr("rows", "3"), ui.Attr("cols", "24"),
				ui.Text("the first line\nanother one here")),
		),
	)
	pages.Run(d, 420, 200)
}

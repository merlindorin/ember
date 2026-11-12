// Panels sized by their items, centred in a flex box, each with a plate
// pulled up into its border by a negative margin larger than the plate:
// as a flex column (stretched and centred) and in block flow, against
// Chrome. Every panel ends its padding and border below its last item.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; color: #f5ffe8; font-size: 12px; line-height: 16px; }
body { margin: 0; height: 220px; display: flex; align-items: center; justify-content: center; gap: 16px; }
.panel { border: 16px solid #6b4a3a; padding: 8px 8px 1px 8px; background-color: #e8d5b0; }
.column { display: flex; flex-direction: column; }
.centred { display: flex; flex-direction: column; align-items: center; }
.plate { display: flex; align-items: center; width: max-content; height: 16px; margin: -24px auto 5px auto;
  padding-bottom: 1px; border-style: solid; border-color: #b0303a; border-width: 0 16px; background-color: #d84a4a; }
.plate.high { margin-top: -40px; margin-bottom: 21px; }
.tabs { display: flex; justify-content: center; gap: 2px; margin-bottom: 3px; }
.tab { height: 16px; width: 40px; background-color: #3a6ab0; }
.rows { height: 32px; width: 96px; margin: 0 auto; background-color: #6cc35a; }
.back { display: flex; justify-content: center; margin-top: 4px; }
.button { width: 48px; height: 16px; background-color: #e8c547; }
`

func panel(kind string, plate ...string) *ui.Node {
	return ui.Div(ui.Class("panel", kind),
		ui.Div(ui.Class(append([]string{"plate"}, plate...)...), ui.Text("Title")),
		ui.Div(ui.Class("tabs"), ui.Div(ui.Class("tab")), ui.Div(ui.Class("tab"))),
		ui.Div(ui.Class("rows")),
		ui.Div(ui.Class("back"), ui.Div(ui.Class("button"))))
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(panel("column"), panel("centred"), panel("centred", "high"), panel("block"))
	pages.Run(d, 640, 220)
}

// Two modal dialogs over positioned boxes with z-index 10 and 99: the top
// layer stacks over every z-index, the dialog shown modal last highest,
// though it comes first in the tree, each over its own backdrop.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; color: #f4ede4; }
body { padding: 12px; }
.badge { position: relative; width: 120px; height: 60px; margin-bottom: 8px; padding: 8px; background-color: #ad5a2c; }
.ten { z-index: 10; }
.top { z-index: 99; background-color: #2c7aad; position: absolute; left: 150px; top: 40px; }
dialog { position: fixed; margin: 0; padding: 12px; border: 1px solid #ad9773; background-color: #2b2722;
  color: #f4ede4; }
.under { left: 60px; top: 50px; width: 200px; height: 120px; }
.over { left: 110px; top: 130px; width: 160px; height: 70px; }
.inner { position: relative; z-index: 1; width: 60px; height: 24px; background-color: #6b8f3a; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	over := ui.Dialog(ui.Class("over"), ui.Text("shown modal last"))
	under := ui.Dialog(ui.Class("under"), ui.Text("shown modal first"), ui.Div(ui.Class("inner")))
	d.Body().Append(
		over,
		ui.Div(ui.Class("badge", "ten"), ui.Text("z-index 10")),
		ui.Div(ui.Class("badge", "top"), ui.Text("z-index 99")),
		under,
	)
	d.ShowModal(under)
	d.ShowModal(over)
	pages.Run(d, 320, 240)
}

// The controls paint draws marks for, enabled above and disabled below,
// in the user agent's own looks: checkbox, radios, range, switch and
// select, as Chrome greys them.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #ffffff; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 16px; align-items: center; margin-bottom: 12px; }
`

func controls(off bool) *ui.Node {
	arg := func(args ...ui.Arg) []ui.Arg {
		if off {
			args = append(args, ui.Attr("disabled", ""))
		}
		return args
	}
	return ui.Div(ui.Class("row"),
		ui.Input(arg(ui.Type("checkbox"), ui.Attr("checked", ""))...),
		ui.Input(arg(ui.Type("checkbox"))...),
		ui.Input(arg(ui.Type("radio"), ui.Attr("checked", ""))...),
		ui.Input(arg(ui.Type("radio"))...),
		ui.Input(arg(ui.Type("range"), ui.Value("30"))...),
		ui.Input(arg(ui.Type("checkbox"), ui.Attr("switch", ""), ui.Attr("checked", ""))...),
		ui.Select(arg(ui.Option(ui.Text("Walls")), ui.Option(ui.Text("Floor")))...),
	)
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(controls(false), controls(true))
	pages.Run(d, 480, 110)
}

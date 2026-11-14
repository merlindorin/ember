// 02 RichText: every tag and variant of kit.RichText on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 220px;
  gap: 10px 16px; align-items: baseline; }
.box { border: 1px dashed #5a524a; }
`

const long = "The [b]quick[/b] brown [i]fox[/i] jumps over the [color=#ffce76]lazy dog[/color] at dusk"

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, rich *ui.Node) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), ui.Div(ui.Class("box"), rich))
	}
	row("plain", kit.RichText("Health: 100"))
	row("bold", kit.RichText("Press [b]Enter[/b] to start"))
	row("italic", kit.RichText("A [i]rusty[/i] key"))
	row("color", kit.RichText("Gold: [color=#ffce76]250[/color] coins"))
	row("named color", kit.RichText("[color=tomato]Burning[/color] for 3 turns"))
	row("nested", kit.RichText("[b]Bold and [i]both[/i] and [color=#8fd18f]green[/color][/b]"))
	row("crossed", kit.RichText("[b]a [i]b[/b] c[/i] d"))
	row("unknown", kit.RichText("[u]kept[/u] as typed"))
	row("wrap", kit.RichText(long, ui.Class("kit-wrap")))
	row("truncate", kit.RichText(long, ui.Class("kit-truncate")))
	row("center", kit.RichText("[b]centred[/b]", ui.Class("kit-center")))
	row("right", kit.RichText("[i]right[/i]", ui.Class("kit-right")))
	pages.Run(d, 380, 470)
}

// 17 Choice: kit.Choice selected, with nothing selected, segmented, hovered and focused, and
// as colour swatches, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 1fr;
  gap: 14px 16px; align-items: center; }
.swatch { display: block; width: 18px; height: 18px; border-radius: 3px; }
`

func labels(names ...string) []*ui.Node {
	var out []*ui.Node
	for _, n := range names {
		out = append(out, kit.Label(n))
	}
	return out
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, n *ui.Node) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), ui.Div(n))
	}
	row("selected", kit.Choice(1, labels("Tiles", "Objects", "Lights")))
	row("none", kit.Choice(-1, labels("Small", "Medium", "Large")))
	row("segmented", kit.Choice(0, labels("Day", "Dusk", "Night"), ui.Class("kit-segmented")))
	hovered := kit.Choice(0, labels("Walls", "Floors", "Props"))
	row("hovered", hovered)
	pages.Pin(d, hovered.Children()[2], dom.Hover)
	focused := kit.Choice(0, labels("North", "East", "South", "West"))
	row("keyboard focus", focused)
	pages.Pin(d, focused.Children()[0], dom.Focus|dom.FocusVisible)
	var swatches []*ui.Node
	for _, c := range []string{"#c0543c", "#d9a441", "#5f9d58", "#4a7bb7", "#8a5fb0"} {
		swatches = append(swatches, ui.Span(ui.Class("swatch"), ui.Style("background-color: "+c)))
	}
	row("swatches", kit.Choice(2, swatches))
	row("disabled item", kit.Choice(0, []*ui.Node{kit.Label("Open"), kit.ChoiceItem(kit.Label("Locked"), ui.Disabled())}))
	pages.Run(d, 380, 330)
}

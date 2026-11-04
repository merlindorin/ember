// 18 PopUp: kit.PopUp showing an item or its placeholder, hovered,
// pressed and focused, on ember's window colour. Its open list is
// ember's own, Chrome showing none in a screenshot.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px 24px; display: grid;
  grid-template-columns: max-content max-content; gap: 12px 24px; align-items: center; }
`

func main() {
	shapes := []string{"Rectangle", "Ellipse", "Polygon"}
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, p *ui.Node, state dom.State) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), p)
		if state != 0 {
			pages.Pin(d, p, state)
		}
	}
	row("default", kit.PopUp(1, shapes), 0)
	row("placeholder", kit.PopUp(-1, shapes, ui.Placeholder("Shape")), 0)
	row("hover", kit.PopUp(0, shapes), dom.Hover)
	row("pressed", kit.PopUp(0, shapes), dom.Active)
	row("focus-visible", kit.PopUp(2, shapes), dom.Focus|dom.FocusVisible)
	pages.Run(d, 320, 230)
}

// 12 Slider: kit.Slider at its ends and between, hovered, focused and
// over a Range, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px 24px; display: grid;
  grid-template-columns: max-content max-content; gap: 14px 24px; align-items: center; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, s *ui.Node, state dom.State) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), s)
		if state != 0 {
			pages.Pin(d, s, state)
		}
	}
	row("0", kit.Slider(0), 0)
	row("30", kit.Slider(30), 0)
	row("100", kit.Slider(100), 0)
	row("hover", kit.Slider(60), dom.Hover)
	row("focus-visible", kit.Slider(45), dom.Focus|dom.FocusVisible)
	row("range 2–8 at 5", kit.Slider(5, kit.Range(2, 8), kit.Step(1)), 0)
	row("disabled", kit.Slider(40, ui.Disabled()), 0)
	pages.Run(d, 360, 296)
}

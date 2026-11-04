// 16 NumberField: kit.NumberField with whole and half steps, empty with a placeholder,
// in every state, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid;
  grid-template-columns: repeat(4, max-content); gap: 12px 16px; align-items: center; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(kit.Label(""), kit.Label("step 1", ui.Class("kit-caps")), kit.Label("step 0.5", ui.Class("kit-caps")),
		kit.Label("empty", ui.Class("kit-caps")))
	row := func(name string, state dom.State, args ...ui.Arg) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")))
		fields := []*ui.Node{
			kit.NumberField(12, append([]ui.Arg{kit.Range(0, 64), kit.Step(1)}, args...)...),
			kit.NumberField(0.5, append([]ui.Arg{kit.Range(0, 1), kit.Step(0.5)}, args...)...),
			kit.NumberField(0, append([]ui.Arg{ui.Value(""), ui.Placeholder("Height")}, args...)...),
		}
		for _, f := range fields {
			d.Body().Append(f)
			if state != 0 {
				pages.Pin(d, f, state)
			}
		}
	}
	row("default", 0)
	row("hover", dom.Hover)
	row("focus-visible", dom.Focus|dom.FocusVisible)
	row("read-only", 0, kit.ReadOnly())
	row("disabled", 0, ui.Disabled())
	pages.Run(d, 420, 260)
}

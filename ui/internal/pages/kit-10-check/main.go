// 10 Check: kit.Check unchecked and checked in every state, and beside a button on one baseline, on ember's
// window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid;
  grid-template-columns: max-content repeat(2, max-content); gap: 12px 24px; align-items: center; }
.beside { grid-column: 1 / -1; display: grid; grid-template-columns: max-content max-content max-content;
  gap: 12px 24px; align-items: baseline; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(kit.Label(""), kit.Label("unchecked", ui.Class("kit-caps")), kit.Label("checked",
		ui.Class("kit-caps")))
	for _, s := range []struct {
		name  string
		state dom.State
	}{
		{"default", 0},
		{"hover", dom.Hover},
		{"focus-visible", dom.Focus | dom.FocusVisible},
		{"disabled", dom.Disabled},
	} {
		d.Body().Append(kit.Label(s.name, ui.Class("kit-form-label")))
		for _, on := range []bool{false, true} {
			args := []ui.Arg{ui.Checked(on)}
			if s.state == dom.Disabled {
				args = append(args, ui.Disabled())
			}
			c := kit.Check("Solid tile", args...)
			d.Body().Append(c)
			switch s.state {
			case dom.Hover:
				pages.Pin(d, c, dom.Hover)
			case dom.Focus | dom.FocusVisible:
				pages.Pin(d, kit.CheckBox(c), s.state)
			// States this page pins on the check itself.
			case dom.Focus, dom.FocusVisible, dom.Active, dom.Disabled, dom.Checked, dom.Open, dom.Modal:
			}
		}
	}
	d.Body().Append(ui.Div(ui.Class("beside"), kit.Label("beside a button", ui.Class("kit-form-label")),
		kit.Check("Shadows"), kit.Button("Reset")))
	pages.Run(d, 360, 240)
}

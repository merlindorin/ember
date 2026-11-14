// 14 TextInput: kit.TextInput with a value, a placeholder, as a password
// and a search field, in every state, on ember's window colour.
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
  grid-template-columns: max-content repeat(2, max-content); gap: 12px 16px; align-items: center; }
.kit-text-input { width: 170px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(kit.Label(""), kit.Label("value", ui.Class("kit-caps")), kit.Label("placeholder",
		ui.Class("kit-caps")))
	row := func(name string, state dom.State, args ...ui.Arg) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")))
		for _, v := range []string{"Heron", ""} {
			in := kit.TextInput(v, append([]ui.Arg{ui.Placeholder("Tile name")}, args...)...)
			d.Body().Append(in)
			if state != 0 {
				pages.Pin(d, in, state)
			}
		}
	}
	row("default", 0)
	row("hover", dom.Hover)
	row("focus", dom.Focus)
	row("focus-visible", dom.Focus|dom.FocusVisible)
	row("read-only", 0, kit.ReadOnly())
	row("disabled", 0, ui.Disabled())
	row("password", 0, kit.Password())
	d.Body().Append(kit.Label("search", ui.Class("kit-form-label")), kit.TextInput("crypt", kit.Search()),
		kit.TextInput("", kit.Search()))
	pages.Run(d, 470, 370)
}

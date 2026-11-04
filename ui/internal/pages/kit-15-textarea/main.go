// 15 TextArea: kit.TextArea with wrapped text, a placeholder, overflowing
// text that scrolls, Rows and NoWrap, in every state, on ember's window
// colour.
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
  grid-template-columns: max-content repeat(2, max-content); gap: 12px 16px; align-items: baseline; }
.kit-text-area { width: 180px; }
`

const notes = "The crypt's west wing floods at dusk. Torches gutter in the draught."

const long = "Line one\nLine two\nLine three\nLine four\nLine five\nLine six\nLine seven"

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(kit.Label(""), kit.Label("text", ui.Class("kit-caps")), kit.Label("placeholder", ui.Class("kit-caps")))
	row := func(name string, state dom.State, args ...ui.Arg) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")))
		for _, v := range []string{notes, ""} {
			a := kit.TextArea(v, append([]ui.Arg{ui.Placeholder("Notes"), kit.Rows(3)}, args...)...)
			d.Body().Append(a)
			if state != 0 {
				pages.Pin(d, a, state)
			}
		}
	}
	row("default", 0)
	row("hover", dom.Hover)
	row("focus-visible", dom.Focus|dom.FocusVisible)
	row("read-only", 0, kit.ReadOnly())
	row("disabled", 0, ui.Disabled())
	d.Body().Append(kit.Label("scrolls", ui.Class("kit-form-label")), kit.TextArea(long, kit.Rows(3)), kit.TextArea(notes,
		kit.Rows(2), kit.NoWrap()))
	pages.Run(d, 560, 550)
}

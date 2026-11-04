// 14b Search: kit.SearchField with text, its clear button showing, empty with its
// placeholder and no clear button, focused and hovered, on ember's window colour.
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
  grid-template-columns: max-content max-content; gap: 12px 16px; align-items: center; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name, text string, state dom.State) {
		f := kit.SearchField(text)
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), f)
		if state != 0 {
			pages.Pin(d, kit.SearchInput(f), state)
		}
	}
	row("with text", "crypt walls", 0)
	row("empty", "", 0)
	row("hover", "props", dom.Hover)
	row("focus-visible", "torch", dom.Focus|dom.FocusVisible)
	pages.Run(d, 340, 200)
}

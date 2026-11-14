// 09 Button: kit.Button, default, primary and destructive, in every state,
// on ember's window colour.
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
  grid-template-columns: max-content repeat(3, max-content); gap: 12px 16px; align-items: center; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(kit.Label(""), kit.Label("default", ui.Class("kit-caps")), kit.Label("primary", ui.Class("kit-caps")),
		kit.Label("destructive", ui.Class("kit-caps")))
	states := []struct {
		name  string
		state dom.State
	}{
		{"default", 0},
		{"hover", dom.Hover},
		{"focus-visible", dom.Focus | dom.FocusVisible},
		{"active", dom.Hover | dom.Active},
		{"disabled", dom.Disabled},
	}
	for _, s := range states {
		d.Body().Append(kit.Label(s.name, ui.Class("kit-form-label")))
		for _, class := range []string{"", "kit-primary", "kit-destructive"} {
			args := []ui.Arg{}
			if class != "" {
				args = append(args, ui.Class(class))
			}
			if s.state == dom.Disabled {
				args = append(args, ui.Disabled())
			}
			b := kit.Button("Save tileset", args...)
			d.Body().Append(b)
			if s.state != 0 && s.state != dom.Disabled {
				pages.Pin(d, b, s.state)
			}
		}
	}
	pages.Run(d, 470, 240)
}

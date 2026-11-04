// 11 Switch: kit.Switch off and on in every state, on ember's
// window colour.
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
  grid-template-columns: max-content repeat(2, max-content); gap: 12px 24px; align-items: center; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(kit.Label(""), kit.Label("off", ui.Class("kit-caps")), kit.Label("on", ui.Class("kit-caps")))
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
			c := kit.Switch("Music", args...)
			d.Body().Append(c)
			switch s.state {
			case dom.Hover:
				pages.Pin(d, c, dom.Hover)
			case dom.Focus | dom.FocusVisible:
				pages.Pin(d, kit.SwitchTrack(c), s.state)
			// States this page pins on the switch itself.
			case dom.Focus, dom.FocusVisible, dom.Active, dom.Disabled, dom.Checked, dom.Open, dom.Modal:
			}
		}
	}
	pages.Run(d, 360, 200)
}

// Pinned states: one button per state, so a still picture shows each in
// ember and in Chrome.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: flex; gap: 12px; }
button { padding: 6px 12px; border: 1px solid var(--border-color); border-radius: 4px;
  background-color: var(--button-color); color: var(--light-text); }
button:hover { background-color: var(--hover-color); color: var(--accent-color); }
button:active { background-color: var(--active-color); }
button:focus-visible { outline: 2px solid var(--accent-color); outline-offset: 2px; }
button:disabled { color: var(--muted-color); }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"states.css": {Data: []byte(sheet)}}, "states.css"))
	for _, s := range []struct {
		text  string
		state dom.State
	}{{"default", 0}, {"hover", dom.Hover}, {"active", dom.Hover | dom.Active}, {"focus", dom.Focus | dom.FocusVisible}} {
		b := ui.Button(ui.Text(s.text))
		d.Body().Append(b)
		if s.state != 0 {
			pages.Pin(d, b, s.state)
		}
	}
	d.Body().Append(ui.Button(ui.Text("disabled"), ui.Disabled()))
	pages.Run(d, 520, 80)
}

// 08 Panel: kit.Panel plain, titled, checked, glowing black and accent, collapsible open and folded,
// and kit.Card at rest, hovered and focused, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 240px;
  gap: 14px 16px; align-items: baseline; }
.inside { padding: 8px; }
.halo { margin: 14px 0; }
`

func inside(text string) *ui.Node { return kit.Label(text, ui.Class("inside")) }

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, n *ui.Node) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), n)
	}
	row("plain", kit.Panel(inside("Panel content")))
	row("title", kit.Panel(kit.Title("Settings"), inside("Volume, language")))
	row("checked", kit.Panel(ui.Class("kit-checked"), inside("Selected slot")))
	row("glow", kit.Panel(ui.Class("kit-glow", "halo"), inside("Floating panel")))
	row("glow accent 18", kit.Panel(ui.Class("halo"), kit.Glow("rgba(240, 192, 96, 0.6)", 18), inside("Glow(accent, 18)")))
	row("collapsible open", kit.CollapsiblePanel("Lights", true, inside("Torch radius 64")))
	row("collapsible folded", kit.CollapsiblePanel("Shadows", false, inside("hidden")))
	row("card", kit.Card(kit.Label("New game")))
	hovered := kit.Card(kit.Label("Continue"))
	row("card hover", hovered)
	focused := kit.Card(kit.Label("Settings"))
	row("card focus", focused)
	pages.Pin(d, hovered, dom.Hover)
	pages.Pin(d, focused, dom.FocusVisible)
	pages.Run(d, 420, 720)
}

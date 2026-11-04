// 23 Tabs: kit.Tabs of a settings window with its second tab chosen and another hovered,
// and a view-toggle Choice, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: flex; flex-direction: column; gap: 20px; }
.kit-tabs { width: 400px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	tabs := kit.Tabs(
		kit.Tab("Video", kit.Label("Resolution, VSync and scale")),
		kit.Tab("Audio", kit.Label("Master, music and effects volumes")),
		kit.Tab("Controls", kit.Label("Key bindings")),
		kit.Tab("Game", kit.Label("Difficulty and language")),
	)
	toggle := kit.Choice(0, []*ui.Node{kit.Label("List"), kit.Label("Grid")}, ui.Class("kit-segmented", "kit-view-toggle"))
	d.Body().Append(tabs, ui.Div(toggle))
	kit.SetSelectedTab(d, tabs, 1)
	pages.Pin(d, kit.TabOf(tabs, 2), dom.Hover)
	pages.Run(d, 440, 200)
}

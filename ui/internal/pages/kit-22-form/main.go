// 22 Form: kit.Form of a text input, a slider, a separator and a check, then a second form
// lined up with it by SetLabelWidth, its fields one width by SetFieldWidth, on ember's
// window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: flex; flex-direction: column; gap: 18px; }
.kit-form { width: 320px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	player := kit.Form(
		kit.Field("Name", kit.TextInput("Ayla")),
		kit.Field("Volume", kit.Slider(60, kit.Range(0, 100))),
		kit.Separator(ui.Class("kit-divider")),
		kit.Field("Music", kit.Check("On", ui.Checked(true))),
	)
	tile := kit.Form(
		kit.Field("Width", kit.NumberField(32)),
		kit.Field("Height in tiles", kit.NumberField(2)),
		kit.Field("Seed", kit.TextInput("crypt-7")),
	)
	for _, f := range []*ui.Node{player, tile} {
		kit.SetLabelWidth(f, 110)
	}
	kit.SetFieldWidth(tile, 120)
	d.Body().Append(player, tile)
	pages.Run(d, 360, 280)
}

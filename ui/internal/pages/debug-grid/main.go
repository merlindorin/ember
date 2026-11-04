// A debug panel's grid of slots, as the demo once laid it out: titles over
// four rows of a number, a weapon, a NumberField filling a 64px track and
// three right-aligned figures, the first two rows' fields enabled; rows on
// their baseline beside rows centred.
package main

import (
	"fmt"
	"strconv"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 12px; display: flex; gap: 12px; align-items: flex-start;
  font-size: 13px; }
.panel { width: 300px; display: flex; flex-direction: column; gap: 4px; }
.panel input { font-size: 13px; }
.fill { width: 100%; }
.slots { column-gap: 12px; row-gap: 4px; }
.centred { align-items: center; }
`

func slots(class string) *ui.Node {
	g := kit.Grid(6, kit.Columns("max-content 1fr 64px repeat(3, max-content)"), ui.Class("fill", "slots", class))
	for _, h := range []string{"Slot", "Weapon", "Level", "DPS", "Hits", "Kills"} {
		g.Append(kit.Label(h, kit.FormLabel()))
	}
	for i := range 4 {
		level := kit.NumberField(float64(1+i%3), kit.Range(1, 5), kit.Step(1), ui.Class("fill"))
		if i >= 2 {
			level.SetAttr("disabled", "")
		}
		g.Append(kit.Label(strconv.Itoa(i+1)), kit.Label(fmt.Sprint("Bow ", i)), level, kit.Label("12", kit.Right()),
			kit.Label("3", kit.Right()), kit.Label("1", kit.Right()))
	}
	return ui.Div(ui.Class("panel"), kit.Label("Slots", kit.FormLabel()), g)
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(slots(""), slots("centred"))
	pages.Run(d, 640, 180)
}

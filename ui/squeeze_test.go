package ui_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

func TestAGridOfFieldsAtTheTopOfAnOverflowingColumnKeepsItsRows(t *testing.T) {
	sheet := `.body { display: flex; flex-direction: column; height: 150px; overflow-y: auto; gap: 4px; width: 300px; }
.slots { column-gap: 12px; row-gap: 4px; }
.fill { width: 100%; }`
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"p.css": {Data: []byte(sheet)}}, "p.css"))
	g := kit.Grid(3, kit.Columns("max-content 1fr 64px"), ui.Class("slots"))
	var fields []*ui.Node
	for i := range 4 {
		f := kit.NumberField(float64(i), ui.Class("fill"))
		g.Append(kit.Label(fmt.Sprint(i)), kit.Label("Bow"), f)
		fields = append(fields, f)
	}
	d.Body().Append(ui.Div(ui.Class("body"), g, ui.Div(ui.Style("height: 400px; flex: none"))))
	d.Layout(geom.V2(640, 360))
	for i := 1; i < len(fields); i++ {
		above, below := fields[i-1].Box(), fields[i].Box()
		assert.InDelta(t, above.Y+above.H+4, below.Y, 0.01,
			"row %d starts a gap under the field above, the grid not squeezed to its labels", i)
	}
}

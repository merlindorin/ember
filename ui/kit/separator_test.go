package kit_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestSeparatorsTakeEmberUIsSlots(t *testing.T) {
	d := ui.NewDocument(kit.Theme())
	menu := kit.Separator()
	divider := kit.Separator(ui.Class("kit-divider"))
	col := ui.Div(ui.Style("width: 100px"), kit.Label("Open"), menu, kit.Label("Save"), divider)
	vertical := kit.Separator(ui.Class("kit-vertical"))
	row := ui.Div(ui.Style("display: flex; height: 30px"), kit.Label("Cut"), vertical, kit.Label("Paste"))
	d.Body().Append(col, row)
	d.Layout(geom.V2(200, 200))
	assert.Empty(t, d.Warnings())
	open := col.Children()[0].Box()
	assert.Equal(t, geom.Rect{X: 6, Y: open.Y + open.H + 4, W: 88, H: 1}, menu.Box(),
		"a 1-point line inset 6, in a 9-point slot")
	save := col.Children()[2].Box()
	assert.Equal(t, menu.Box().Y+1+4, save.Y)
	assert.Equal(t, float32(100), divider.Box().W, "a divider runs across")
	assert.Equal(t, float32(1), vertical.Box().W)
	assert.Equal(t, float32(15), vertical.Box().H, "a line of the theme's 15-point text tall")
}

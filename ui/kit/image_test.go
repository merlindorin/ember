package kit_test

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestAColorRectFillsItsBox(t *testing.T) {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"t.css": {Data: []byte(".swatch { flex: 1 }")}}, "t.css"))
	r := kit.ColorRect("#ffce76", ui.Class("swatch"))
	d.Body().Append(ui.Div(ui.Style("width: 40px; height: 20px; display: flex"), r))
	d.Layout(geom.V2(200, 100))
	assert.Empty(t, d.Warnings())
	assert.Equal(t, float32(40), r.Box().W)
	assert.Equal(t, float32(20), r.Box().H, "stretched by its flex parent")
	assert.Equal(t, []string{"kit-color-rect", "swatch"}, r.Classes())
}

func TestANinePatchKeepsItsMarginsAndHoldsItsChildrenInside(t *testing.T) {
	files := fstest.MapFS{"t.css": {Data: []byte(".narrow { width: 0 }")}}
	panel, err := os.ReadFile("../../examples/22-nine-patch/panel.png")
	if err != nil {
		t.Fatal(err)
	}
	files["panel.png"] = &fstest.MapFile{Data: panel}
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(files, "t.css"))
	child := kit.Label("inside")
	p := kit.NinePatch("panel.png", 8, 6, 10, 4, child)
	empty := kit.NinePatch("panel.png", 8, 8, 8, 8, ui.Class("narrow"))
	d.Body().Append(ui.Div(ui.Style("width: 120px"), p), ui.Div(empty))
	d.Layout(geom.V2(200, 100))
	assert.Equal(t, float32(8), child.Box().X-p.Box().X, "left margin")
	assert.Equal(t, float32(6), child.Box().Y-p.Box().Y, "top margin")
	assert.Equal(t, float32(120), p.Box().W)
	assert.Equal(t, 6+child.Box().H+4, p.Box().H, "its child and the top and bottom margins")
	assert.Equal(t, float32(16), empty.Box().H, "never smaller than its margins")
	assert.Equal(t, float32(16), empty.Box().W)
	assert.Empty(t, d.Warnings(), "the image is found")
}

func TestAnImageIsAsLargeAsItsPictureUnlessStretched(t *testing.T) {
	panel, err := os.ReadFile("../../examples/22-nine-patch/panel.png")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{"t.css": {Data: []byte(".wide { width: 60px; height: 30px; }")}, "panel.png": {Data: panel}}
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(files, "t.css"))
	natural := kit.Image("panel.png")
	wide := kit.Image("panel.png", ui.Class("wide", "kit-keep-aspect-centered"))
	tile := kit.TileImage("panel.png", ui.Class("wide"))
	d.Body().Append(ui.Div(natural), ui.Div(wide), tile)
	d.Layout(geom.V2(200, 200))
	assert.Empty(t, d.Warnings())
	assert.Equal(t, geom.Vec2{X: 24, Y: 24}, geom.Vec2{X: natural.Box().W, Y: natural.Box().H}, "its picture's size")
	assert.Equal(t, geom.Vec2{X: 60, Y: 30}, geom.Vec2{X: wide.Box().W, Y: wide.Box().H})
	assert.Equal(t, float32(60), tile.Box().W)
	assert.Equal(t, []string{"kit-image", "wide", "kit-keep-aspect-centered"}, wide.Classes())
}

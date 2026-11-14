package main

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// walker is a sprite of 20-sprite-sheet and its tag: its name over its
// health bar, which follows it 20 points above its centre.
type walker struct {
	sprite *twod.Sprite
	tag    *ui.Node
	bar    *ui.Node
	health float64
	route  [2]geom.Vec2
	toward int
}

func newWalkers(sheet *twod.Texture) []*walker {
	frames := twod.Strip(sheet, twod.Cell(16, 16))
	clips := twod.NewAnimations().
		Add("down", 8, frames[0:4], twod.Loop()).
		Add("up", 8, frames[4:8], twod.Loop()).
		Add("side", 8, frames[8:12], twod.Loop())
	var out []*walker
	for _, w := range []struct {
		name  string
		route [2]geom.Vec2
	}{
		{"Ada", [2]geom.Vec2{geom.V2(120, 120), geom.V2(260, 120)}},
		{"Bo", [2]geom.Vec2{geom.V2(400, 90), geom.V2(400, 300)}},
		{"Cy", [2]geom.Vec2{geom.V2(160, 290), geom.V2(520, 230)}},
	} {
		sprite := twod.NewSprite(sheet, twod.Animate(clips), twod.At(w.route[0].X, w.route[0].Y), twod.Scale(2, 2))
		bar := kit.Progress(1, kit.Range(0, 1), kit.ShowValue(false))
		tag := ui.Div(ui.Class("tag"), ui.Follow(sprite, geom.V2(0, -20)), kit.Label(w.name), bar)
		out = append(out, &walker{sprite: sprite, tag: tag, bar: bar, health: 1, route: w.route, toward: 1})
	}
	return out
}

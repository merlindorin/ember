// Blend modes and masks. A night layer multiplies the meadow dark blue;
// a lantern, a layer used as a mask, shows the daylit meadow only where
// its soft disc is, following the mouse; fireflies add their light to
// whatever is under them.
//
//	go run ./examples/31-blend-modes
package main

import (
	"embed"
	"log"
	"math/rand/v2"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed tiles.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · blend modes"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)

	sheet, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	tiles := twod.NewTileset(sheet, twod.Cell(16, 16), twod.TileAnimation(water, 2, water, ripple))
	meadow := func() *twod.TileMap {
		return twod.NewTileMap(tiles, twod.ASCII(level, legend()), twod.CellSize(32, 32))
	}

	ground := twod.NewLayer(0)
	ground.Add(meadow())

	night := twod.NewLayer(1, twod.Blend(twod.Multiply))
	night.Add(twod.NewPolygon(
		[]geom.Vec2{geom.V2(0, 0), geom.V2(640, 0), geom.V2(640, 384), geom.V2(0, 384)},
		twod.Color(color.RGB(0.12, 0.14, 0.32)),
	))

	light := glow(128)
	lantern := twod.NewLayer(3, twod.AsMask())
	flame := twod.NewSprite(light, twod.At(320, 192), twod.Scale(3, 3))
	lantern.Add(flame)

	day := twod.NewLayer(2, twod.MaskedBy(lantern))
	day.Add(meadow())

	fireflies := twod.NewLayer(4)
	random := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a game's randomness, not a secret
	for range 24 {
		fireflies.Add(twod.NewSprite(light,
			twod.At(640*random.Float32(), 384*random.Float32()),
			twod.Scale(0.25, 0.25),
			twod.Tint(color.RGB(1, 0.8, 0.3)),
			twod.Blend(twod.Add),
		))
	}

	win.Add(ground, night, day, lantern, fireflies)

	ember.New().
		AddEntities(win).
		AddSystems(&Lantern{light: flame, keys: win.Input()}).
		MustRun()
}

// Shape lights: night in the garden of 28-y-sort, lit only by what glows.
// A pool of lava warms the grass round it, a neon tube hangs by the trees,
// a crystal throws the shadows of the pillars, and a rune shines on the
// ground. The hero has no light of his own: he is lit as he walks past them.
//
//	go run ./examples/40-shape-lights
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
)

//go:embed tiles.png objects.png walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · shape lights"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)

	ground, things, err := loadTilesets()
	if err != nil {
		log.Fatal(err)
	}
	grassland := twod.NewTileMap(ground, twod.ASCII(field, map[rune]int{'.': grass}), twod.CellSize(32, 32))

	floor := twod.NewLayer(-1, light.Lit()).Add(grassland)
	garden := twod.NewLayer(0, twod.SortByY(), light.Lit())
	hero, err := newHero(grassland.Center(10, 7))
	if err != nil {
		log.Fatal(err)
	}
	crystal := newCrystal(grassland.Center(8, 4))
	garden.Add(hero,
		newLava(grassland.Center(16, 10)),
		newNeon(grassland.Center(3, 10)),
		crystal,
		newRune(grassland.Center(9, 10)),
	)
	trees := twod.NewTileMap(things,
		twod.ASCII(objects, objectsLegend()),
		twod.CellSize(32, 32),
		twod.TallIn(garden),
	)

	space := collide.NewSpace()
	space.AddSolid(trees)
	night := twod.NewCamera(twod.At(320, 192), light.Ambient(color.RGB(0.08, 0.09, 0.16)), light.OccludersFrom(space))
	win.Add(floor, garden, trees, night)

	ember.New().
		AddEntities(win).
		AddSystems(
			collide.NewTracker(space),
			&Walk{hero: hero, keys: win.Input(), space: space},
			&Keys{crystal: crystal, in: win.Input()},
		).
		MustRun()
}

// loadTilesets cuts the grass from tiles.png and the trees and pillars,
// tall and solid, from objects.png.
func loadTilesets() (*twod.Tileset, *twod.Tileset, error) {
	sheet, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		return nil, nil, err
	}
	tall, err := twod.LoadTexture(files, "objects.png", twod.Pixelated())
	if err != nil {
		return nil, nil, err
	}
	return twod.NewTileset(sheet, twod.Cell(16, 16)),
		twod.NewTileset(tall, twod.Cell(16, 32), twod.Tall(tree, pillar), twod.Solid(tree, pillar)),
		nil
}

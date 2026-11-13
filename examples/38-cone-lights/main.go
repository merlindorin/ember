// Cone lights: night in the garden of 28-y-sort. The hero carries a
// flashlight that points at the mouse, and the trees and pillars cast
// shadows out of its cone. F switches it off and on.
//
//	go run ./examples/38-cone-lights
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
)

//go:embed tiles.png objects.png walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · cone lights"),
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
	hero, err := newHero(grassland.Center(10, 6), light.Unlit())
	if err != nil {
		log.Fatal(err)
	}
	beam := newFlashlight()
	hero.Add(beam)
	garden.Add(hero)
	trees := twod.NewTileMap(things,
		twod.ASCII(objects, objectsLegend()),
		twod.CellSize(32, 32),
		twod.TallIn(garden),
	)

	space := collide.NewSpace()
	space.AddSolid(tiles.Of(trees))
	night := twod.NewCamera(twod.At(320, 192), light.Ambient(color.RGB(0.08, 0.09, 0.16)), light.OccludersFrom(space))
	win.Add(floor, garden, trees, night)

	ember.New().
		AddEntities(win).
		AddSystems(
			collide.NewTracker(space),
			&Walk{hero: hero, keys: win.Input(), space: space},
			&Flashlight{beam: beam, camera: night, in: win.Input()},
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
		twod.NewTileset(tall, twod.Cell(16, 32), twod.Tall(tree, pillar), tiles.Solid(tree, pillar)),
		nil
}

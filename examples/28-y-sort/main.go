// Y-sorting: trees and pillars are tall tiles. They stand as sprites at
// their feet in the garden, a layer that draws its children by their y, so
// the hero shows behind a tree when north of it and in front when south.
// Their trunks are solid: the hero walks behind the canopies, not through
// the trees.
//
//	go run ./examples/28-y-sort
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed tiles.png objects.png walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · y-sort"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)

	ground, things, err := loadTilesets()
	if err != nil {
		log.Fatal(err)
	}
	grassland := twod.NewTileMap(ground, twod.ASCII(field, map[rune]int{'.': grass}), twod.CellSize(32, 32))

	garden := twod.NewLayer(0, twod.SortByY())
	hero, err := newHero(grassland.Center(10, 6))
	if err != nil {
		log.Fatal(err)
	}
	garden.Add(hero)
	trees := twod.NewTileMap(things,
		twod.ASCII(objects, objectsLegend()),
		twod.CellSize(32, 32),
		twod.TallIn(garden),
	)

	space := collide.NewSpace()
	space.AddSolid(trees)
	win.Add(grassland, garden, trees)

	ember.New().
		AddEntities(win).
		AddSystems(
			collide.NewTracker(space),
			&Walk{hero: hero, keys: win.Input(), space: space},
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

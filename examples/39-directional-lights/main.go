// Directional lights: the garden of 28-y-sort under a sun that crosses the sky
// every 12 seconds. The trees and pillars throw long shadows at dawn and
// dusk, short ones at noon, and the night falls for 4 seconds in between.
//
//	go run ./examples/39-directional-lights
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
)

//go:embed tiles.png objects.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · directional lights"),
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
	trees := twod.NewTileMap(things,
		twod.ASCII(objects, objectsLegend()),
		twod.CellSize(32, 32),
		twod.TallIn(garden),
	)

	camera := twod.NewCamera(twod.At(320, 192), light.Ambient(moonlight()))
	sun := newSun()
	win.Add(floor, garden, trees, camera, sun)

	ember.New().
		AddEntities(win).
		AddSystems(&Day{sun: sun, camera: camera, in: win.Input()}).
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

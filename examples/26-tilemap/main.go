// A tile map: a tileset cuts tiles.png into numbered tiles, and the map is
// drawn from a picture of the level, one character per cell. Water is an
// animated tile: every water cell plays its two frames in step.
//
//	go run ./examples/26-tilemap
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed tiles.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · tile map"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)

	sheet, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	tiles := twod.NewTileset(sheet, twod.Cell(16, 16),
		twod.TileAnimation(water, 2, water, ripple),
	)
	meadow := twod.NewTileMap(tiles,
		twod.ASCII(level, legend()),
		twod.CellSize(32, 32),
	)
	win.Add(meadow)

	ember.New().
		AddEntities(win).
		MustRun()
}

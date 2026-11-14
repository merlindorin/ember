// Tiled pictures: the garden of 28-y-sort under two pictures the game
// paints a cell at a time. Fog of war hides what the hero hasn't seen and
// lifts as he walks; an ambient map lights a clearing at the garden's
// heart, darkening toward its edges, under his torch.
//
//	go run ./examples/42-tiled-pictures
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
)

//go:embed tiles.png objects.png walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · tiled pictures"),
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
	hero, err := newHero(grassland.Center(10, 6))
	if err != nil {
		log.Fatal(err)
	}
	torch := newTorch()
	hero.Add(torch)
	garden.Add(hero)
	trees := twod.NewTileMap(things,
		twod.ASCII(objects, objectsLegend()),
		twod.CellSize(32, 32),
		twod.TallIn(garden),
	)

	fog := newFog()
	above := twod.NewLayer(1).Add(fog.pic)

	space := collide.NewSpace()
	space.AddSolid(tiles.Of(trees))
	camera := twod.NewCamera(twod.At(320, 192), light.AmbientMap(newShade()), light.OccludersFrom(space))
	win.Add(floor, garden, trees, above, camera)

	ember.New().
		AddEntities(win).
		AddSystems(
			collide.NewTracker(space),
			&Walk{hero: hero, keys: win.Input(), space: space},
			&Flicker{torch: torch},
			&Reveal{fog: fog, hero: hero},
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

// Probes: night in the garden of 28-y-sort, with slimes lurking in the
// dark. The hero carries his torch, and a probe on each slime reads the
// light where it stands: in the dark only its eyes show, and when the
// torchlight reaches it, the whole slime does.
//
//	go run ./examples/41-probes
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
)

//go:embed tiles.png objects.png walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · probes"),
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
	torch := newTorch()
	hero.Add(torch)
	garden.Add(hero)

	lurk := &Lurk{}
	for _, cell := range [][2]int{{7, 4}, {12, 4}, {4, 8}, {16, 6}, {14, 10}, {2, 2}} {
		s := newSlime(grassland.Center(cell[0], cell[1]))
		garden.Add(s.body, s.eyes, s.probe)
		lurk.slimes = append(lurk.slimes, s)
	}

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
			&Flicker{torch: torch},
			lurk,
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

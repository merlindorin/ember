// Shadows: night in the garden of 28-y-sort, lit by a lamp that follows
// the mouse. The trees and pillars stand 64 points tall and the crates 24,
// so with the lamp 100 points up, the crates' shadows end soon while the
// trees' run on. Up and down raise and lower the lamp, S gives it a size
// and F fades its shadows.
//
//	go run ./examples/37-shadows
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

//go:embed tiles.png objects.png crate.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · shadows"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)

	ground, things, err := loadTilesets()
	if err != nil {
		log.Fatal(err)
	}
	crate, err := twod.LoadTexture(files, "crate.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	grassland := twod.NewTileMap(ground, twod.ASCII(field, map[rune]int{'.': grass}), twod.CellSize(32, 32))

	floor := twod.NewLayer(-1, light.Lit()).Add(grassland)
	garden := twod.NewLayer(0, twod.SortByY(), light.Lit())
	for _, cell := range [][2]int{{4, 4}, {11, 8}, {16, 2}, {7, 10}} {
		at := grassland.Center(cell[0], cell[1])
		garden.Add(twod.NewSprite(crate, twod.At(at.X, at.Y), twod.Scale(2, 2), light.Occluder(), light.Height(12)))
	}
	trees := twod.NewTileMap(things,
		twod.ASCII(objects, objectsLegend()),
		twod.CellSize(32, 32),
		twod.TallIn(garden),
	)

	lamp := newLamp()
	label := twod.NewText("", twod.At(12, 10), twod.Pivot(0, 0))
	hud := twod.NewLayer(1, twod.Fixed()).Add(label)

	space := collide.NewSpace()
	space.AddSolid(trees)
	camera := twod.NewCamera(twod.At(320, 192), light.Ambient(color.RGB(0.08, 0.09, 0.16)), light.OccludersFrom(space))
	win.Add(floor, garden, trees, hud, camera, lamp)

	ember.New().
		AddEntities(win).
		AddSystems(&Lamp{lamp: lamp, camera: camera, in: win.Input(), label: label}).
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

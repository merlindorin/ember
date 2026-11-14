// Input: the hero walks with the arrows or WASD through an action map, a
// ghost tree follows the pointer into the world, a click plants it there,
// and the wheel zooms. The bindings print as JSON, ready to save.
//
//	go run ./examples/49-actions
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed walker.png tiles.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · input"),
		platform.Size(640, 400),
		platform.FixedSize(),
		platform.Background(color.Hex("#5b9a4a")),
	)
	win.SetCursor(platform.CursorHidden)

	tiles, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	hero, err := newHero()
	if err != nil {
		log.Fatal(err)
	}
	garden := newGarden(tiles)
	eye := twod.NewCamera(twod.Follow(hero), twod.Smoothing(6))

	win.Add(
		twod.NewLayer(0, twod.SortByY()).Add(garden.trees, hero),
		twod.NewLayer(1).Add(garden.ghost),
		twod.NewLayer(2, twod.Fixed()).Add(
			twod.NewText(
				"WASD or arrows to walk · click or Enter to plant · wheel to zoom",
				twod.At(16, 12), twod.Pivot(0, 0), twod.FontSize(15),
			),
		),
		eye,
	)

	controls := newControls()
	saved, err := json.Marshal(controls)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, string(saved))

	ember.New().
		AddEntities(win).
		AddSystems(
			&Walk{hero: hero, in: win.Input(), controls: controls},
			&Plant{garden: garden, eye: eye, in: win.Input(), controls: controls},
		).
		MustRun()
}

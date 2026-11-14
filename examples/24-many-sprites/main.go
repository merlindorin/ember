// Many sprites: a bunnymark. Five hundred bunnies bounce around the window,
// and holding Space or the mouse button adds a hundred more every frame;
// the counter shows how many there are and the frame rate.
//
//	go run ./examples/24-many-sprites
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed bunny.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · many sprites"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)

	bunny, err := twod.LoadTexture(files, "bunny.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	field := twod.NewLayer(0)
	counter := twod.NewText("", twod.At(12, 10))
	hud := twod.NewLayer(1)
	hud.Add(counter)
	win.Add(field, hud)

	ember.New().
		AddEntities(win).
		AddSystems(newBunnies(bunny, win, field, counter)).
		MustRun()
}

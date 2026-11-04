// A 2D camera: it follows the hero along a level wider than the window,
// easing behind him and stopping at the level's ends. The far hills move at
// half its speed, and the HUD stays put.
//
//	go run ./examples/23-2d-camera
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed walker.png
var files embed.FS

// level is the world the camera may show: wider than the window, as tall.
func level() geom.Rect {
	return geom.Rect{W: 1600, H: 360}
}

func main() {
	win := platform.NewWindow(
		platform.Title("ember · 2D camera"),
		platform.Size(640, 360),
		platform.FixedSize(),
		platform.Background(color.Hex("#9cc7e8")),
	)

	hero, err := newHero()
	if err != nil {
		log.Fatal(err)
	}

	win.Add(
		newHills(),
		twod.NewLayer(0).Add(newGround(), hero),
		twod.NewLayer(1, twod.Fixed()).Add(
			twod.NewText("← → to walk", twod.At(16, 12), twod.Pivot(0, 0)),
		),
		twod.NewCamera(
			twod.Follow(hero),
			twod.Smoothing(6),
			twod.Bounds(level()),
		),
	)

	ember.New().
		AddEntities(win).
		AddSystems(&Walk{hero: hero, keys: win.Input()}).
		MustRun()
}

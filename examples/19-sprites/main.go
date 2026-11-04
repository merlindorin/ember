// Sprites: a PNG loaded from the example's files, drawn three times. A
// sprite is an entity made with options: where it is, how it turns and
// scales, whether it is mirrored and how it is tinted. Each is the
// window's child, so it shows in it.
//
//	go run ./examples/19-sprites
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed crate.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · sprites"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)

	crate, err := twod.LoadTexture(files, "crate.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}

	plain := twod.NewSprite(crate,
		twod.At(160, 180),
		twod.Scale(6, 6),
	)
	turned := twod.NewSprite(crate,
		twod.At(320, 180),
		twod.Scale(8, 8),
		twod.Rotation(0.4),
	)
	mossy := twod.NewSprite(crate,
		twod.At(480, 180),
		twod.Scale(6, 6),
		twod.Flip(true, false),
		twod.Tint(color.Hex("#8fd18a")),
	)
	win.Add(plain, turned, mossy)

	ember.New().
		AddEntities(win).
		MustRun()
}

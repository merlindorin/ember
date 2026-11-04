// A sprite sheet: one PNG holds every frame of a walking character, a row
// per direction. Strip cuts it into frames, Animations names the clips,
// and the sprite plays them. Walk moves it with the arrow keys.
//
//	go run ./examples/20-sprite-sheet
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed walker.png tiles.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · sprite sheet"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)

	sheet, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	tiles, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	frames := twod.Strip(sheet, twod.Cell(16, 16))
	clips := twod.NewAnimations().
		Add("down", 8, frames[0:4], twod.Loop()).
		Add("up", 8, frames[4:8], twod.Loop()).
		Add("side", 8, frames[8:12], twod.Loop())

	hero := twod.NewSprite(sheet,
		twod.Animate(clips),
		twod.At(320, 180),
		twod.Scale(5, 5),
	)
	win.Add(newGround(tiles), hero)

	ember.New().
		AddEntities(win).
		AddSystems(&Walk{hero: hero, keys: win.Input()}).
		MustRun()
}

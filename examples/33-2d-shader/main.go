// A 2D shader: click the meadow to make ripples. The world layer runs an
// Effect, a WGSL function that pushes its pixels out in rings, and the game
// tells it where and when each ripple started by setting its uniforms
// every frame. The help text is on a layer of its own, so it stays still.
//
//	go run ./examples/33-2d-shader
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/twod"
)

//go:embed tiles.png walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · 2D shader"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)

	sheet, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	walker, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}

	ripples := render.NewEffect(rippleShader, render.Name("ripples"))
	meadow := twod.NewLayer(0, twod.WithEffect(ripples))
	meadow.Add(
		twod.NewTileMap(twod.NewTileset(sheet, twod.Cell(16, 16)), twod.ASCII(level, legend()), twod.CellSize(32, 32)),
		twod.NewSprite(
			walker,
			twod.Still(twod.Strip(walker, twod.Cell(16, 16))[0]),
			twod.At(300, 240),
			twod.Scale(3, 3),
		),
	)

	help := twod.NewLayer(1, twod.Fixed())
	help.Add(twod.NewText("Click to make ripples", twod.At(16, 12), twod.Pivot(0, 0)))

	win.Add(meadow, help)

	ember.New().
		AddEntities(win).
		AddSystems(&Ripples{effect: ripples, window: win}).
		MustRun()
}

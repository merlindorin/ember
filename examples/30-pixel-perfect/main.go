// Pixel-perfect scaling: the camera draws the world at 320×180 pixels,
// then scales it up by whole factors to fill the window, letterboxed, so
// every pixel of the art stays square and sharp whatever the window's
// size. The hint is on a fixed layer: it draws at the window's resolution,
// so its text stays smooth.
//
//	go run ./examples/30-pixel-perfect
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed tiles.png walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · pixel perfect"),
		platform.Size(1000, 600),
		platform.Background(color.RGB(0.015, 0.012, 0.02)),
	)

	sheet, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	tiles := twod.NewTileset(sheet, twod.Cell(16, 16), twod.TileAnimation(water, 2, water, ripple))
	meadow := twod.NewTileMap(tiles, twod.ASCII(level, legend()))
	win.Add(meadow)

	walker, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	frames := twod.Strip(walker, twod.Cell(16, 16))
	clips := twod.NewAnimations().
		Add("down", 8, frames[0:4], twod.Loop()).
		Add("up", 8, frames[4:8], twod.Loop()).
		Add("side", 8, frames[8:12], twod.Loop())
	hero := twod.NewSprite(walker, twod.Animate(clips), twod.At(150, 120))
	win.Add(hero)

	cols, rows := meadow.Columns(), meadow.Rows()
	win.Add(twod.NewCamera(
		twod.Resolution(320, 180),
		twod.Follow(hero),
		twod.Bounds(geom.Rect{W: float32(cols * 16), H: float32(rows * 16)}),
	))

	hud := twod.NewLayer(1, twod.Fixed())
	hud.Add(twod.NewText("arrows to walk · resize the window", twod.At(16, 16), twod.Pivot(0, 0)))
	win.Add(hud)

	ember.New().
		AddEntities(win).
		AddSystems(&Walk{hero: hero, keys: win.Input()}).
		MustRun()
}

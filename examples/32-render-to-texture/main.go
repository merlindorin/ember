// Render to texture: a second camera draws the whole meadow into a render
// texture, and a sprite on the HUD shows that texture as a minimap. The
// window's own camera follows the hero up close; the minimap camera is not
// one of the window's, it only feeds its texture.
//
//	go run ./examples/32-render-to-texture
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
		platform.Title("ember · render to texture"),
		platform.Size(960, 540),
		platform.FixedSize(),
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
	size := geom.V2(float32(cols*16), float32(rows*16))
	win.Add(twod.NewCamera(
		twod.Zoom(3),
		twod.Follow(hero),
		twod.Bounds(geom.Rect{W: size.X, H: size.Y}),
		twod.SnapToPixels(),
	))

	minimap := twod.NewRenderTexture(240, 128, twod.Pixelated())
	win.Add(twod.NewCamera(
		twod.Into(minimap),
		twod.Zoom(0.5),
		twod.At(size.X/2, size.Y/2),
	))

	hud := twod.NewLayer(1, twod.Fixed())
	hud.Add(
		twod.NewPolygon(
			[]geom.Vec2{geom.V2(0, 0), geom.V2(248, 0), geom.V2(248, 136), geom.V2(0, 136)},
			twod.At(700, 12),
			twod.Color(color.RGB(0.02, 0.02, 0.03)),
		),
		twod.NewSprite(minimap, twod.At(824, 80)),
	)
	win.Add(hud)

	ember.New().
		AddEntities(win).
		AddSystems(&Walk{hero: hero, keys: win.Input()}).
		MustRun()
}

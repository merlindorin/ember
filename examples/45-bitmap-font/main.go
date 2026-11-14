// Bitmap font: text drawn from a picture of its characters, pixel for
// pixel, for a crisp retro look. The engine's own bitmap font, and a game's
// arcade font loaded from a PNG, showing a score that counts up.
//
//	go run ./examples/45-bitmap-font
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed arcade.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · bitmap font"),
		platform.Size(640, 360),
		platform.FixedSize(),
		platform.Background(color.Hex("#14121f")),
	)

	arcade, err := twod.LoadBitmapFont(files, "arcade.png", twod.Cell(6, 8), twod.Charset(" 0123456789SCOREHILV"))
	if err != nil {
		log.Fatal(err)
	}

	hello := twod.NewBitmapText("The engine's own bitmap font",
		twod.At(320, 60),
		twod.Scale(2, 2),
		twod.TextShadow(color.Hex("#3b3355")),
	)
	score := twod.NewBitmapText("SCORE 000000",
		twod.Glyphs(arcade),
		twod.At(320, 170),
		twod.Scale(5, 5),
		twod.Color(color.Hex("#ffd166")),
		twod.TextShadow(color.Hex("#a3441f")),
	)
	high := twod.NewBitmapText("HI 019000",
		twod.Glyphs(arcade),
		twod.At(320, 250),
		twod.Scale(3, 3),
		twod.Color(color.Hex("#7fdbca")),
	)
	lives := twod.NewBitmapText("LIVES 3",
		twod.Glyphs(arcade),
		twod.At(320, 300),
		twod.Scale(3, 3),
		twod.Color(color.Hex("#ef476f")),
	)
	win.Add(hello, score, high, lives)

	ember.New().
		AddEntities(win).
		AddSystems(&Score{text: score}).
		MustRun()
}

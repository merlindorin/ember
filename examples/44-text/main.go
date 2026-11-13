// Text: words drawn with a typeface at a size and a colour. The engine's
// faces, a face read from a font file, a paragraph wrapped to a width, and
// a clock whose text changes every frame.
//
//	go run ./examples/44-text
package main

import (
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

const story = "The keeper lit the lamp at dusk, as every evening, " +
	"and the ships far out at sea turned their bows toward the light."

func main() {
	win := platform.NewWindow(
		platform.Title("ember · text"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)

	pixel, err := loadPixelFont()
	if err != nil {
		log.Fatal(err)
	}

	title := twod.NewText("The Lighthouse",
		twod.At(24, 20),
		twod.Pivot(0, 0),
		twod.Typeface(twod.Bold()),
		twod.FontSize(32),
	)
	faces := twod.NewLayer(0).Add(
		twod.NewText("Regular", twod.At(24, 76), twod.Pivot(0, 0)),
		twod.NewText("Bold", twod.At(120, 76), twod.Pivot(0, 0), twod.Typeface(twod.Bold())),
		twod.NewText("Italic", twod.At(186, 76), twod.Pivot(0, 0), twod.Typeface(twod.Italic())),
		twod.NewText("Small caps", twod.At(250, 76), twod.Pivot(0, 0), twod.Typeface(twod.SmallCaps())),
		twod.NewText("Mono", twod.At(370, 76), twod.Pivot(0, 0), twod.Typeface(twod.Mono())),
		twod.NewText("Silkscreen", twod.At(440, 76), twod.Pivot(0, 0), twod.Typeface(pixel)),
	)
	paragraph := twod.NewText(story,
		twod.At(24, 124),
		twod.Pivot(0, 0),
		twod.Wrap(300),
		twod.FontSize(18),
		twod.Color(color.Hex("#f2e6c9")),
	)
	clock := twod.NewText("",
		twod.At(400, 300),
		twod.Pivot(0, 0),
		twod.Typeface(pixel),
		twod.FontSize(24),
		twod.Color(color.Hex("#e8a33d")),
	)

	win.Add(title, faces, paragraph, clock)

	ember.New().
		AddEntities(win).
		AddSystems(&Clock{text: clock}).
		MustRun()
}

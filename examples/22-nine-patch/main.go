// Nine-patch: one small framed panel stretched to any size. Its corners are
// kept as they are, its edges stretch one way and its middle both ways, so
// a dialog, a button, a bar and a pillar all come from the same 24×24 PNG.
//
//	go run ./examples/22-nine-patch
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed panel.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · nine-patch"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)

	panel, err := twod.LoadTexture(files, "panel.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}

	dialog := twod.NewNinePatch(panel, twod.Margins(8, 8, 8, 8),
		twod.Size(64, 48),
		twod.At(40, 60),
		twod.Pivot(0, 0),
		twod.Scale(3, 3),
	)
	button := twod.NewNinePatch(panel, twod.Margins(8, 8, 8, 8),
		twod.Size(48, 16),
		twod.At(300, 60),
		twod.Pivot(0, 0),
		twod.Scale(3, 3),
	)
	pillar := twod.NewNinePatch(panel, twod.Margins(8, 8, 8, 8),
		twod.Size(16, 56),
		twod.At(520, 60),
		twod.Pivot(0, 0),
		twod.Scale(3, 3),
	)
	bar := twod.NewNinePatch(panel, twod.Margins(8, 8, 8, 8),
		twod.Size(184, 16),
		twod.At(40, 260),
		twod.Pivot(0, 0),
		twod.Scale(3, 3),
	)
	win.Add(dialog, button, pillar, bar)

	ember.New().
		AddEntities(win).
		MustRun()
}

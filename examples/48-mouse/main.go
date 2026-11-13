// Mouse: crates on a floor, dragged with the left button wherever the
// camera looks. The pointer is in window points; the camera turns it into
// the world under it, through its zoom, which the wheel changes. The hand
// cursor shows over a crate, a picture cursor of the example's own while
// one is carried, and the Back and Forward buttons undo and redo the last
// moves.
//
//	go run ./examples/48-mouse
package main

import (
	"bytes"
	"embed"
	"image"
	"image/png"
	"io/fs"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed crate.png move.png move@2x.png
var files embed.FS

// moveCursor is the four-way arrow shown while a crate is carried, its hot
// spot in its middle, with its picture at twice the size for HiDPI
// screens.
func moveCursor() platform.Cursor {
	small, big := picture("move.png"), picture("move@2x.png")
	return platform.NewCursor(small, 12, 12, platform.HiDPI(big))
}

func main() {
	win := platform.NewWindow(
		platform.Title("ember · mouse"),
		platform.Size(640, 360),
		platform.FixedSize(),
		platform.Background(color.Hex("#3a3f4b")),
	)
	crate, err := twod.LoadTexture(files, "crate.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	crates := []*twod.Sprite{
		twod.NewSprite(crate, twod.At(-120, 0), twod.Size(64, 64)),
		twod.NewSprite(crate, twod.At(0, 0), twod.Size(64, 64)),
		twod.NewSprite(crate, twod.At(120, 0), twod.Size(64, 64)),
	}
	eye := twod.NewCamera()
	floor := twod.NewLayer(0)
	for _, c := range crates {
		floor.Add(c)
	}

	win.Add(
		floor,
		twod.NewLayer(1, twod.Fixed()).Add(
			twod.NewText(
				"drag a crate · wheel to zoom · Back and Forward to undo and redo",
				twod.At(16, 12), twod.Pivot(0, 0), twod.FontSize(15),
			),
		),
		eye,
	)

	ember.New().
		AddEntities(win).
		AddSystems(&Drag{window: win, eye: eye, crates: crates, carry: moveCursor()}).
		MustRun()
}

func picture(name string) image.Image {
	data, err := fs.ReadFile(files, name)
	if err != nil {
		log.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	return img
}

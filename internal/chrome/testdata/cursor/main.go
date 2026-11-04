//go:build js

// Command cursor shows a picture cursor with a HiDPI picture over its
// canvas, for TestAPictureCursorShowsOnTheCanvas.
package main

import (
	"image"
	"image/color"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
)

func main() {
	small, big := image.NewNRGBA(image.Rect(0, 0, 16, 16)), image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := range 16 {
		small.SetNRGBA(i, i, color.NRGBA{255, 255, 255, 255})
		big.SetNRGBA(2*i, 2*i, color.NRGBA{255, 255, 255, 255})
	}
	win := platform.NewWindow(platform.Size(320, 200), platform.Title("cursor"))
	win.SetCursor(platform.NewCursor(small, 3, 5, platform.HiDPI(big)))
	ember.New().AddEntities(win).MustRun()
}

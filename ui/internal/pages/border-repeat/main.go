// One patterned frame laid by each border-image-repeat: stretch, repeat,
// round and space, and the two axes apart, at sizes that aren't whole
// copies, against Chrome's border-image.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; }
body { padding: 12px; margin: 0; display: flex; flex-wrap: wrap; align-items: flex-start; gap: 12px; }
.f { border-style: solid; border-color: transparent; border-width: 8px; border-image: url("frame.png") 8 fill;
  width: 45px; height: 29px; }
.wide { width: 101px; height: 13px; }
.stretch { border-image-repeat: stretch; }
.repeat { border-image-repeat: repeat; }
.round { border-image-repeat: round; }
.space { border-image-repeat: space; }
.mixed { border-image-repeat: round repeat; }
`

// frame is a 24×24 nine-patch: gold 8×8 corners, edges striped across
// their length two pixels at a time, and a checkered centre, so each
// copy's start shows.
func frame() []byte {
	gold, dark := color.RGBA{0xe8, 0xb8, 0x4a, 0xff}, color.RGBA{0x3a, 0x24, 0x18, 0xff}
	red, blue := color.RGBA{0xc8, 0x40, 0x30, 0xff}, color.RGBA{0x30, 0x60, 0xc0, 0xff}
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := range 24 {
		for x := range 24 {
			edgeX, edgeY := x >= 8 && x < 16, y >= 8 && y < 16
			var c color.RGBA
			switch {
			case edgeX && edgeY:
				c = red
				if (x/2+y/2)%2 == 1 {
					c = blue
				}
			case edgeX:
				c = gold
				if (x-8)%4 >= 2 || x == 8 {
					c = dark
				}
			case edgeY:
				c = gold
				if (y-8)%4 >= 2 || y == 8 {
					c = dark
				}
			default:
				c = gold
			}
			img.SetRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "frame.png": {Data: frame()}},
		"page.css"))
	for _, how := range []string{"stretch", "repeat", "round", "space", "mixed"} {
		d.Body().Append(ui.Div(ui.Class("f", how)))
	}
	for _, how := range []string{"repeat", "round", "space"} {
		d.Body().Append(ui.Div(ui.Class("f", "wide", how)))
	}
	pages.Run(d, 420, 130)
}

// One patterned frame drawn at each border-image-width: the border's
// width, thicker and thinner, a number, a length, a percentage and auto,
// sides that don't fit the box, and a band with no top or bottom, against
// Chrome's border-image.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; }
body { padding: 12px; margin: 0; display: flex; flex-wrap: wrap; align-items: flex-start; gap: 12px; }
.f { border-style: solid; border-color: transparent; border-width: 8px; border-image: url("frame.png") 8 fill;
  width: 45px; height: 29px; }
.thin { border-width: 4px; }
.thick { border-image-width: 16px; }
.thick-repeat { border-image-width: 16px; border-image-repeat: repeat; }
.number { border-width: 3px; border-image-width: 2; }
.pct { border-image-width: 25%; width: 64px; height: 40px; }
.auto { border-width: 2px; border-image-width: auto; }
.shrink { border-image-width: 30px; width: 24px; height: 8px; }
.band { border-width: 0 8px; width: 101px; height: 4px; }
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
	for _, how := range []string{"plain", "thin", "thick", "thick-repeat", "number", "pct", "auto", "shrink", "band"} {
		d.Body().Append(ui.Div(ui.Class("f", how)))
	}
	pages.Run(d, 460, 170)
}

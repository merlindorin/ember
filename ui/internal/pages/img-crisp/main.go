// Pixel-art pictures scaled by a non-integer number of points, at
// positions off the point grid: a pixelated img beside the same picture as
// a pixelated background. Chrome puts each on whole device pixels, its
// texels the same width.
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
html { background-color: #ffffff; }
body { margin: 0; }
img, .bg { position: absolute; width: 24px; height: 24px; image-rendering: pixelated; }
.bg { background-image: url(checker.png); background-size: 24px 24px; }
.a { left: 8.25px; top: 8.25px; }
.b { left: 40.5px; top: 8.5px; }
.c { left: 72.75px; top: 8.75px; }
.d { left: 104.4px; top: 8.6px; width: 40px; height: 40px; }
.bg.a { top: 56.25px; } .bg.b { top: 56.5px; } .bg.c { top: 56.75px; }
.bg.d { top: 56.6px; background-size: 40px 40px; }
`

// checker is a 16-pixel picture whose pixels alternate red and blue.
func checker() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			c := color.RGBA{255, 0, 0, 255}
			if (x+y)%2 == 1 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "checker.png": {Data: checker()}},
		"page.css"))
	for _, at := range []string{"a", "b", "c", "d"} {
		d.Body().Append(ui.Img(ui.Attr("src", "checker.png"), ui.Class(at)), ui.Div(ui.Class("bg", at)))
	}
	pages.Run(d, 160, 104)
}

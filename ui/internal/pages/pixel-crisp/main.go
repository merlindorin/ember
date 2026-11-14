// Pixel crispness at a whole Document zoom (EMBER_UI_ZOOM, 1 by default):
// a 16px tile of a one-pixel checker as a pixelated background, the pixel
// HUD's four one-pixel rims, and Silkscreen at 8px with a one-pixel
// outline of four text shadows, smoothed and then unsmoothed
// (-webkit-font-smoothing: none). At whole zooms each texel, rim and glyph
// pixel should land on whole device pixels in flat colour.
package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed Silkscreen-Regular.ttf
var silkscreen []byte

const sheet = `
@font-face { font-family: Pix; src: url(Silkscreen-Regular.ttf); }
html { background-color: #ffffff; }
body { margin: 0; }
.tile { position: absolute; left: 8px; top: 8px; width: 64px; height: 32px; background-image: url(checker.png);
  image-rendering: pixelated; }
.rims { position: absolute; left: 100px; top: 12px; width: 24px; height: 24px; background-color: #222222;
  box-shadow: 0 0 0 1px #3b2412, 0 0 0 2px #d9a44e, 0 0 0 3px #7a5428, 0 0 0 4px #0b0706; }
.words.px { top: 80px; -webkit-font-smoothing: none; }
.words { position: absolute; left: 8px; top: 56px; padding: 4px; background-color: #334455; color: #ffffff;
  font-family: Pix; font-size: 8px; line-height: 8px;
    text-shadow: 1px 0 0 #000, -1px 0 0 #000, 0 1px 0 #000, 0 -1px 0 #000; }
`

// checker is a 16-pixel tile whose pixels alternate red and blue.
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
	zoom := 1
	if z, err := strconv.Atoi(os.Getenv("EMBER_UI_ZOOM")); err == nil && z > 0 {
		zoom = z
	}
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}, "checker.png": {Data: checker()},
		"Silkscreen-Regular.ttf": {Data: silkscreen}}
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"), ui.Zoom(float32(zoom)))
	d.Body().Append(ui.Div(ui.Class("tile")), ui.Div(ui.Class("rims")), ui.Div(ui.Class("words"),
		ui.Text("ZOMBIE 128 HP")),
		ui.Div(ui.Class("words", "px"), ui.Text("ZOMBIE 128 HP")))
	pages.Run(d, 160*zoom, 104*zoom)
}

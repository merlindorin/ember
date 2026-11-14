// Boxes off the pixel grid under a whole Document zoom (EMBER_UI_ZOOM, 3 by
// default): each is flex-centred in a column, 47.5 by 48.5 points in 64
// by 101, so it lands on quarter points: at zoom 3 and 2 pixels a point,
// on half device pixels, as a dialog centred in a window whose height
// isn't a multiple of the zoom does. Chrome snaps box
// edges to device pixels, keeping a pixelated border-image's bands, a
// border and a background crisp; ember draws them on the same pixels.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; }
body { margin: 0; padding: 0 8px; display: flex; gap: 8px; }
.column { height: 101px; width: 64px; display: flex; flex-direction: column;
  justify-content: center; align-items: center; }
.panel { width: 47.5px; height: 48.5px; border-style: solid; border-color: transparent; border-width: 6px;
  border-image: url("frame.png") 6 fill; image-rendering: pixelated; }
.bordered { width: 47.5px; height: 48.5px; border: 3px solid #c9a36b; background-color: #2b2722; }
.filled { width: 47.5px; height: 48.5px; background-color: #8f6cf0; }
`

// frame is a 13×13 nine-patch in three bands a side, 2 art pixels each,
// light to dark, round a dark centre: crisp, it shows three bands per
// edge; filtered across two rows, they blend.
func frame() []byte {
	bands := []color.NRGBA{
		{R: 0xd9, G: 0xa4, B: 0x4e, A: 0xff}, {R: 0x7a, G: 0x54, B: 0x28, A: 0xff}, {R: 0x3b, G: 0x24, B: 0x12, A: 0xff},
	}
	img := image.NewNRGBA(image.Rect(0, 0, 13, 13))
	for y := range 13 {
		for x := range 13 {
			d := min(x, y, 12-x, 12-y)
			c := color.NRGBA{R: 0x14, G: 0x0e, B: 0x0b, A: 0xff}
			if d < 6 {
				c = bands[d/2]
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func main() {
	zoom := 3
	if z, err := strconv.Atoi(os.Getenv("EMBER_UI_ZOOM")); err == nil && z > 0 {
		zoom = z
	}
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}, "frame.png": {Data: frame()}}
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"), ui.Zoom(float32(zoom)))
	for _, class := range []string{"panel", "bordered", "filled"} {
		d.Body().Append(ui.Div(ui.Class("column"), ui.Div(ui.Class(class))))
	}
	pages.Run(d, 232*zoom, 101*zoom)
}

// A border-image over a solid, coloured border: where the image loads it
// replaces the border's own sides, so its transparent corners show what
// is under the box, not the border colour; where it doesn't load, the
// solid border is drawn. A div and a button, against Chrome.
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
html { background-color: #1f3a2e; }
body { padding: 12px; margin: 0; display: flex; align-items: flex-start; gap: 12px; }
.framed { border-style: solid; border-width: 8px; border-color: #e63946; border-image: url("frame.png") 8 fill; }
.box { width: 60px; height: 40px; }
button.framed { width: 96px; height: 56px; padding: 0; }
.lost { border-image-source: url("missing.png"); }
`

// frame is a 24×24 nine-patch: its 8×8 corners clear, its edges gold and
// its centre dark.
func frame() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	for y := range 24 {
		for x := range 24 {
			corner := (x < 8 || x >= 16) && (y < 8 || y >= 16)
			edge := x < 8 || x >= 16 || y < 8 || y >= 16
			switch {
			case corner:
			case edge:
				img.SetNRGBA(x, y, color.NRGBA{R: 0xc9, G: 0xa3, B: 0x6b, A: 0xff})
			default:
				img.SetNRGBA(x, y, color.NRGBA{R: 0x2b, G: 0x27, B: 0x22, A: 0xff})
			}
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func main() {
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}, "frame.png": {Data: frame()}}
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"))
	d.Body().Append(
		ui.Div(ui.Class("framed", "box")),
		ui.Button(ui.Class("framed")),
		ui.Div(ui.Class("framed", "box", "lost")),
	)
	pages.Run(d, 320, 100)
}

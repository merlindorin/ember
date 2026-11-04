// Rounded clips (decision 0058): a scrolled list in a rounded panel, a
// picture larger than its rounded frame, a rounded box inside a rounded
// one, and a gradient under rounded corners, each cut to its padding
// box's inner curve as Chrome cuts it.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; color: #f4ede4; }
body { padding: 12px; display: flex; flex-wrap: wrap; gap: 14px; align-items: flex-start; }
.panel { width: 220px; height: 150px; overflow: auto; border: 2px solid #ad9773; border-radius: 16px; }
.row { height: 24px; padding: 0 10px; line-height: 24px; background-color: #3a332c; }
.row:nth-child(odd) { background-color: #5a4630; }
.frame { width: 120px; height: 90px; overflow: hidden; border: 3px solid #f0b545; border-radius: 12px; }
.picture { width: 200px; height: 160px; margin: -20px; background-image: url(checks.png); }
.outer { width: 140px; height: 100px; overflow: hidden; border: 1px solid #9a8f80; border-radius: 22px;
  padding: 15px 20px; background-color: #2b2722; }
.inner { height: 70px; overflow: hidden; border-radius: 10px; }
.fill { height: 200px; background-color: #7fd36a; }
.gradient { width: 160px; height: 70px; border: 2px solid #f2e6cf; border-radius: 18px;
  background: linear-gradient(90deg, #d9534f, #337ab7); }
`

// checks is a 16-pixel checkerboard, every edge a hard one.
func checks() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 200, 160))
	for y := range 160 {
		for x := range 200 {
			c := color.NRGBA{0xe0, 0x67, 0x4f, 0xff}
			if (x/16+y/16)%2 == 0 {
				c = color.NRGBA{0x5e, 0xa8, 0xec, 0xff}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "checks.png": {Data: checks()}},
		"page.css"))
	panel := ui.Div(ui.Class("panel"))
	for i := range 40 {
		panel.Append(ui.Div(ui.Class("row"), ui.Text(fmt.Sprintf("row %d · rounded", i))))
	}
	d.Body().Append(panel,
		ui.Div(ui.Class("frame"), ui.Div(ui.Class("picture"))),
		ui.Div(ui.Class("outer"), ui.Div(ui.Class("inner"), ui.Div(ui.Class("fill")))),
		ui.Div(ui.Class("gradient")))
	dom.Data(panel).Scroll = geom.V2(0, 101)
	pages.Run(d, 560, 330)
}

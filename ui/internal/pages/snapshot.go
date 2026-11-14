package pages

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/export"
	"github.com/lantervale/ember/ui/internal/assets"
)

// snapshots are the canvases' snapshots from ember's capture at path, of d
// laid out at w points wide: each canvas's content box cut from it.
func snapshots(d *ui.Document, path string, w int) export.Option {
	f, err := os.Open(path) //nolint:gosec // the path is the test harness's own, from EMBER_UI_SNAPSHOTS
	if err != nil {
		panic(err)
	}
	defer f.Close()
	capture, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	scale := float32(capture.Bounds().Dx()) / float32(w)
	return export.Snapshots(func(n *ui.Node) ([]byte, bool) {
		b := assets.Tree(d).Of[n]
		if b == nil {
			return nil, false
		}
		r := n.Box()
		at := geom.Rect{X: r.X + b.Border.Left + b.Padding.Left, Y: r.Y + b.Border.Top + b.Padding.Top}
		at.W = r.W - b.Border.Left - b.Border.Right - b.Padding.Left - b.Padding.Right
		at.H = r.H - b.Border.Top - b.Border.Bottom - b.Padding.Top - b.Padding.Bottom
		if at.W <= 0 || at.H <= 0 {
			return nil, false
		}
		return cut(capture, at, scale), true
	})
}

// cut is the PNG of r, in points, cut from capture at scale pixels a point.
func cut(capture image.Image, r geom.Rect, scale float32) []byte {
	px := func(v float32) int { return int(math.Round(float64(v * scale))) }
	from := image.Rect(px(r.X), px(r.Y), px(r.X+r.W), px(r.Y+r.H)).Add(capture.Bounds().Min)
	out := image.NewRGBA(image.Rect(0, 0, from.Dx(), from.Dy()))
	draw.Draw(out, out.Bounds(), capture, from.Min, draw.Src)
	var b bytes.Buffer
	if err := png.Encode(&b, out); err != nil {
		panic(err)
	}
	return b.Bytes()
}

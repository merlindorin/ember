package twod

import (
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
)

// cropMargin is how many transparent pixels a cropped pass keeps around
// what was drawn, so its filtered edges sample as they did uncropped.
const cropMargin = 2

// picture is a layer's pass shown this frame: its picture's six vertices
// from at in list.
type picture struct {
	pass, mask *draw.Pass
	list       *draw.List
	at         int
}

// cropPictures shrinks each picture's pass to what was drawn in it, and
// its picture to match, so a small masked minimap no longer clears and
// composites a whole window. The crop falls on whole pixels of the pass,
// with transparent ones around, so the picture shows exactly as before. A
// masked layer is cropped to its mask, the two sharing one rectangle; a
// layer with effects keeps its whole pass, as they may read the window.
func cropPictures(pictures []picture) {
	cropped := map[*draw.Pass]geom.Rect{}
	for _, p := range pictures {
		if len(p.pass.Effects) > 0 || (p.mask != nil && len(p.mask.Effects) > 0) {
			continue
		}
		drawn := p.pass
		if p.mask != nil {
			drawn = p.mask
		}
		r, ok := cropped[drawn]
		if !ok {
			if r, ok = cropRect(drawn); !ok {
				continue
			}
			crop(drawn, r)
			cropped[drawn] = r
		}
		if p.mask != nil {
			if _, done := cropped[p.pass]; !done {
				crop(p.pass, r)
				cropped[p.pass] = r
			}
		}
		for i := p.at; i < p.at+4; i++ {
			v := &p.list.Vertices[i]
			v.Pos = geom.V2(r.X+v.UV.X*r.W, r.Y+v.UV.Y*r.H)
		}
	}
}

// cropRect is the rectangle of pass, in points, around what was drawn in
// it, on whole pixels of the pass with a margin, its size rounded up to
// cropStep pixels so a picture moving in its layer keeps its target.
func cropRect(pass *draw.Pass) (geom.Rect, bool) {
	if pass.Screen.X <= 0 || pass.Screen.Y <= 0 {
		return geom.Rect{}, false
	}
	kx, ky := float32(pass.Width)/pass.Screen.X, float32(pass.Height)/pass.Screen.Y
	e, ok := pass.List.Extent()
	if !ok {
		return geom.Rect{W: 1 / kx, H: 1 / ky}, true
	}
	x0, x1 := span(e.X*kx, (e.X+e.W)*kx, pass.Width)
	y0, y1 := span(e.Y*ky, (e.Y+e.H)*ky, pass.Height)
	return geom.Rect{X: float32(x0) / kx, Y: float32(y0) / ky, W: float32(x1-x0) / kx, H: float32(y1-y0) / ky}, true
}

// cropStep is what a cropped pass's size is rounded up to, in pixels.
const cropStep = 32

// span is the whole pixels from lo to hi, with the margin, cropStep long
// at least, inside [0, size].
func span(lo, hi float32, size int) (int, int) {
	a := max(int(math.Floor(float64(lo)))-cropMargin, 0)
	b := min(int(math.Ceil(float64(hi)))+cropMargin, size)
	n := min((max(b-a, 1)+cropStep-1)/cropStep*cropStep, size)
	b = a + n
	if b > size {
		a, b = size-n, size
	}
	return a, b
}

// crop makes pass show only r of its surface.
func crop(pass *draw.Pass, r geom.Rect) {
	kx, ky := float32(pass.Width)/pass.Screen.X, float32(pass.Height)/pass.Screen.Y
	pass.List.Shift(geom.V2(-r.X, -r.Y))
	pass.Width, pass.Height = max(int(r.W*kx+0.5), 1), max(int(r.H*ky+0.5), 1)
	pass.Screen = geom.V2(r.W, r.H)
}

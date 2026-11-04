package paint

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/style"
)

// sliceSpan is a part of a slice along one axis: from and to where it is drawn,
// u0 and u1 the texture coordinates there.
type sliceSpan struct{ from, to, u0, u1 float32 }

// clamped is the slice of texels s0 to s1, of a texture n texels long,
// drawn from d0 to d1 as Chrome draws a border-image slice: sampled only
// within itself, so the half texel at each end holds its edge texel's
// colour rather than blending into the next slice. That is a part of
// constant coordinates at each end and the slice's middle between them.
func clamped(d0, d1, s0, s1, n float32, out []sliceSpan) []sliceSpan {
	if d1 <= d0 || s1 <= s0 {
		return out
	}
	edge := (d1 - d0) / (s1 - s0) / 2
	lo, hi := (s0+0.5)/n, (s1-0.5)/n
	if d1-d0 <= 2*edge {
		mid := (s0 + s1) / 2 / n
		return append(out, sliceSpan{d0, d1, mid, mid})
	}
	return append(out,
		sliceSpan{d0, d0 + edge, lo, lo}, sliceSpan{d0 + edge, d1 - edge, lo, hi}, sliceSpan{d1 - edge, d1, hi, hi})
}

// ninePatch draws tex over r cut by slice, its corners unstretched, its
// edges and centre stretched, each slice sampled within itself.
func (p *painter) ninePatch(r geom.Rect, tex *twod.Texture, slice style.Sides[float32], tint color.Color) {
	size := tex.Size()
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	xs := [4]float32{r.X, r.X + slice.Left, r.X + r.W - slice.Right, r.X + r.W}
	ys := [4]float32{r.Y, r.Y + slice.Top, r.Y + r.H - slice.Bottom, r.Y + r.H}
	us := [4]float32{0, slice.Left, size.X - slice.Right, size.X}
	vs := [4]float32{0, slice.Top, size.Y - slice.Bottom, size.Y}
	var cols, rows [9]sliceSpan
	for j := range 3 {
		across := clamped(ys[j], ys[j+1], vs[j], vs[j+1], size.Y, rows[:0])
		for i := range 3 {
			along := clamped(xs[i], xs[i+1], us[i], us[i+1], size.X, cols[:0])
			for _, y := range across {
				for _, x := range along {
					p.s.Image(geom.Rect{X: x.from, Y: y.from, W: x.to - x.from, H: y.to - y.from}, tex,
						geom.Rect{X: x.u0, Y: y.u0, W: x.u1 - x.u0, H: y.u1 - y.u0}, tint)
				}
			}
		}
	}
}

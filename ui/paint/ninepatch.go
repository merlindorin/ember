package paint

import (
	"math"

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

// tiles are where copies of a slice length long go over d0 to d1 by
// repeat, and whether the first or last is cut by the ends: one stretched
// over it all; copies at their length centred on it (Tile); a whole number
// of copies scaled to fill it (Round); or a whole number at their length
// with the rest shared out as equal gaps (Space), none when one won't fit.
// Copies beyond maxTiles are dropped.
func tiles(d0, d1, length float32, repeat style.ImageRepeat, out [][2]float32) ([][2]float32, bool) {
	extent := d1 - d0
	if extent <= 0 || length <= 0 || repeat == style.RepeatStretch {
		return append(out, [2]float32{d0, d1}), false
	}
	switch repeat {
	case style.RepeatRound:
		n := max(float32(math.Round(float64(extent/length))), 1)
		step := extent / n
		for k := float32(0); k < n && len(out) < maxTiles; k++ {
			out = append(out, [2]float32{d0 + k*step, d0 + (k+1)*step})
		}
		return out, false
	case style.RepeatSpace:
		n := float32(math.Floor(float64(extent / length)))
		gap := (extent - n*length) / (n + 1)
		for k := float32(0); k < n && len(out) < maxTiles; k++ {
			at := d0 + gap + k*(length+gap)
			out = append(out, [2]float32{at, at + length})
		}
		return out, false
	case style.RepeatStretch, style.RepeatTile:
	}
	mid := (d0 + d1) / 2
	at := mid - length/2 - float32(math.Ceil(float64((mid-length/2-d0)/length)))*length
	for ; at < d1 && len(out) < maxTiles; at += length {
		out = append(out, [2]float32{at, at + length})
	}
	return out, out[0][0] < d0 || out[len(out)-1][1] > d1
}

// ninePatch draws tex over r cut by slice, its sides widths thick: its
// corners scaled to their widths, its edges and centre stretched or tiled
// along each axis as repeat says (across, then down), each copy sampled
// within its slice and scaled as its edge is. Copies cut by the ends of
// their part are clipped to it.
func (p *painter) ninePatch(
	r geom.Rect, tex *twod.Texture, slice, widths style.Sides[float32], repeat [2]style.ImageRepeat,
	tint color.Color,
) {
	size := tex.Size()
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	xs := [4]float32{r.X, r.X + widths.Left, r.X + r.W - widths.Right, r.X + r.W}
	ys := [4]float32{r.Y, r.Y + widths.Top, r.Y + r.H - widths.Bottom, r.Y + r.H}
	us := [4]float32{0, slice.Left, size.X - slice.Right, size.X}
	vs := [4]float32{0, slice.Top, size.Y - slice.Bottom, size.Y}
	// A copy along the top or bottom edge, or across the centre, is scaled
	// as its edge is drawn thicker or thinner than its slice: the top's
	// scale for the centre across, the left's down.
	scaleAcross := [3]float32{ratio(widths.Top, slice.Top), ratio(widths.Top, slice.Top),
		ratio(widths.Bottom, slice.Bottom)}
	scaleDown := [3]float32{ratio(widths.Left, slice.Left), ratio(widths.Left, slice.Left),
		ratio(widths.Right, slice.Right)}
	// Only the middle column and row are laid by repeat; the corners and
	// the edges' thickness are stretched over theirs.
	lay := func(d, s [4]float32, k int, how style.ImageRepeat, scale float32, out [][2]float32) ([][2]float32, bool) {
		if k != 1 {
			how = style.RepeatStretch
		}
		return tiles(d[k], d[k+1], (s[k+1]-s[k])*scale, how, out)
	}
	var colTiles, rowTiles [][2]float32
	var cols, rows [9]sliceSpan
	for j := range 3 {
		for i := range 3 {
			var cutX, cutY bool
			rowTiles, cutY = lay(ys, vs, j, repeat[1], scaleDown[i], rowTiles[:0])
			colTiles, cutX = lay(xs, us, i, repeat[0], scaleAcross[j], colTiles[:0])
			if cutX || cutY {
				p.s.PushClip(geom.Rect{X: xs[i], Y: ys[j], W: xs[i+1] - xs[i], H: ys[j+1] - ys[j]})
			}
			for _, ty := range rowTiles {
				across := clamped(ty[0], ty[1], vs[j], vs[j+1], size.Y, rows[:0])
				for _, tx := range colTiles {
					along := clamped(tx[0], tx[1], us[i], us[i+1], size.X, cols[:0])
					for _, y := range across {
						for _, x := range along {
							p.s.Image(geom.Rect{X: x.from, Y: y.from, W: x.to - x.from, H: y.to - y.from}, tex,
								geom.Rect{X: x.u0, Y: y.u0, W: x.u1 - x.u0, H: y.u1 - y.u0}, tint)
						}
					}
				}
			}
			if cutX || cutY {
				p.s.PopClip()
			}
		}
	}
}

// ratio is how much thicker an edge drawn w thick is than its slice s,
// 1 for an empty slice.
func ratio(w, s float32) float32 {
	if s <= 0 {
		return 1
	}
	return w / s
}

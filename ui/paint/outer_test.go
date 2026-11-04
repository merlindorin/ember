package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"math"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func outerBox(sh style.Shadow, edit func(s *style.Computed)) *box.Box {
	return el(10, 10, 50, 50, func(s *style.Computed) {
		sh.Color = shade
		s.Shadows = []style.Shadow{sh}
		if edit != nil {
			edit(s)
		}
	})
}

// meshPoints are a gradient's points in the root's space.
func meshPoints(o op) []geom.Vec2 {
	out := make([]geom.Vec2, len(o.Points))
	for i, q := range o.Points {
		out[i] = o.Xform.Xform(q)
	}
	return out
}

func TestABlurredShadowIsCSSsGaussianOfItsBox(t *testing.T) {
	r := paintOne(outerBox(style.Shadow{X: 2, Y: 4, Blur: 8, Spread: 1}, nil))
	assert.Empty(t, r.of("glow"))
	fades := r.of("gradient")
	require.Len(t, fades, 1)
	ps := meshPoints(fades[0])
	lo, hi := ps[0], ps[0]
	for i, q := range ps {
		lo, hi = geom.V2(min(lo.X, q.X), min(lo.Y, q.Y)), geom.V2(max(hi.X, q.X), max(hi.Y, q.Y))
		if roundedDistanceOf(q, nil) >= clipEdge {
			want := 0.5 * blurredInterval(q.X, 11, 63, 4) * blurredInterval(q.Y, 13, 65, 4)
			assert.InDelta(t, want, fades[0].Alpha[i], 1e-4, "at %v", q)
		}
	}
	assert.Equal(t, geom.V2(11-12, 13-12), lo, "the moved, spread box and three sigmas round it")
	assert.Equal(t, geom.V2(63+12, 65+12), hi)
}

// blurredRounded is the Gaussian blur, by sigma, of the w×h box at the
// origin rounded by r, seen at q: summed over the box a quarter point
// at a time.
func blurredRounded(q geom.Vec2, w, h, r, sigma float32) float32 {
	var sum float64
	const d = 0.25
	for y := float32(d / 2); y < h; y += d {
		for x := float32(d / 2); x < w; x += d {
			if !insideRounded(geom.V2(x, y), 0, 0, w, h, r, 0) {
				continue
			}
			dx, dy := float64(x-q.X), float64(y-q.Y)
			sum += math.Exp(-(dx*dx + dy*dy) / (2 * float64(sigma*sigma)))
		}
	}
	return float32(sum * d * d / (2 * math.Pi * float64(sigma*sigma)))
}

func TestARoundedBlurredShadowIsTheBlurOfTheRoundedBox(t *testing.T) {
	r := paintOne(outerBox(style.Shadow{Blur: 8}, rounded(12)))
	fades := r.of("gradient")
	require.Len(t, fades, 1)
	ps := meshPoints(fades[0])
	checked := 0
	for i, q := range ps {
		local := q.Sub(geom.V2(10, 10))
		if local.X > 25 || local.Y > 25 || int(local.X)%3 != 0 || int(local.Y)%3 != 0 ||
			roundedDistanceOf(q, rounded(12)) < clipEdge {
			continue
		}
		checked++
		assert.InDelta(t, 0.5*blurredRounded(local, 50, 50, 12, 4), fades[0].Alpha[i], 0.004, "at %v", local)
	}
	assert.Greater(t, checked, 20)
}

func TestASharpShadowIsStillAFill(t *testing.T) {
	r := paintOne(outerBox(style.Shadow{X: 3}, rounded(6)))
	assert.Empty(t, r.of("gradient"))
	assert.Len(t, r.of("round"), 1)
}

func BenchmarkAGlowingPanelsShadow(b *testing.B) {
	panel := el(10, 10, 240, 30, func(s *style.Computed) {
		rounded(6)(s)
		s.Shadows = []style.Shadow{{Blur: 24, Spread: 4, Color: shade}}
	})
	r := newRecorder()
	clear(shadowMeshes)
	b.ResetTimer()
	for range b.N {
		r.ops = r.ops[:0]
		Paint(r, tree(panel))
	}
	b.ReportMetric(float64(len(r.of("gradient")[0].Points)), "points")
}

func TestABlurredShadowStaysOutsideItsBorderBox(t *testing.T) {
	for _, edit := range []func(*style.Computed){nil, rounded(12)} {
		r := paintOne(outerBox(style.Shadow{X: 3, Y: 6, Blur: 16, Spread: 2}, edit))
		fades := r.of("gradient")
		require.Len(t, fades, 1)
		outside := 0
		for i, q := range meshPoints(fades[0]) {
			d := roundedDistanceOf(q, edit)
			switch {
			case d <= 0:
				assert.Zero(t, fades[0].Alpha[i], "under the box at %v, as Chrome clips it", q)
			case d >= 0.01:
				outside++
				assert.Greater(t, fades[0].Alpha[i], float32(0), "outside at %v", q)
			}
		}
		assert.Greater(t, outside, 50)
	}
}

func roundedDistanceOf(q geom.Vec2, edit func(*style.Computed)) float32 {
	var rd radii
	if edit != nil {
		rd = radii{12, 12, 12, 12}
	}
	return roundedDistance(q, geom.Rect{X: 10, Y: 10, W: 50, H: 50}, rd)
}

func TestTheRoundingLetsTheShadowShowInTheBoxsCorner(t *testing.T) {
	fades := paintOne(outerBox(style.Shadow{Blur: 16, Spread: 2}, rounded(12))).of("gradient")
	require.Len(t, fades, 1)
	for i, q := range meshPoints(fades[0]) {
		if q == geom.V2(10, 10) {
			assert.Greater(t, fades[0].Alpha[i], float32(0.05))
			return
		}
	}
	t.Fatal("no mesh point at the box's corner")
}

func TestASharpSquareShadowIsWhatTheBoxLeavesOfIt(t *testing.T) {
	r := paintOne(outerBox(style.Shadow{X: 6, Y: 4}, nil))
	box := geom.Rect{X: 10, Y: 10, W: 50, H: 50}
	var area float32
	for _, o := range r.of("rect") {
		assert.Zero(t, overlap(o.Rect, box), "nothing under the box: %v", o.Rect)
		area += o.Rect.W * o.Rect.H
	}
	assert.InDelta(t, 50*50-44*46, area, 1e-3)
}

func overlap(a, b geom.Rect) float32 {
	w := min(a.X+a.W, b.X+b.W) - max(a.X, b.X)
	h := min(a.Y+a.H, b.Y+b.H) - max(a.Y, b.Y)
	return max(w, 0) * max(h, 0)
}

func TestAShadowMeshKeepsOnlyThePointsItDraws(t *testing.T) {
	m := shadowMesh(shadowKey{size: geom.V2(300, 160), rounding: radii{12, 12, 12, 12}, shadow: style.Shadow{Blur: 12},
		color: shade, outer: true,
		box: geom.Rect{W: 300, H: 160}, boxRounding: radii{12, 12, 12, 12}})
	used := make([]bool, len(m.points))
	for _, i := range m.indices {
		used[i] = true
	}
	assert.NotContains(t, used, false, "none under the box")
}

func TestASmallRadiusUnderABigBlurStaysRound(t *testing.T) {
	r := paintOne(outerBox(style.Shadow{Blur: 40, Spread: 2}, rounded(3)))
	fades := r.of("gradient")
	require.Len(t, fades, 1)
	ps := meshPoints(fades[0])
	checked := 0
	for i, q := range ps {
		local := q.Sub(geom.V2(10, 10))
		if roundedDistance(q, geom.Rect{X: 10, Y: 10, W: 50, H: 50}, radii{3, 3, 3, 3}) < 0.01 || local.X > 30 ||
			local.Y > 30 || local.X < -30 || local.Y < -30 {
			continue
		}
		want := 0.5 * blurredRounded(local.Add(geom.V2(2, 2)), 54, 54, 5, 20)
		assert.InDelta(t, want, fades[0].Alpha[i], 0.006, "at %v", local)
		checked++
	}
	assert.Greater(t, checked, 30)
}

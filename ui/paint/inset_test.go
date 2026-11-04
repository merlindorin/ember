package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals // the tests' fixed values; Go has no constant colours or pictures
var shade = color.RGBA(0, 0, 0, 0.5)

func insetBox(sh style.Shadow, edit func(s *style.Computed)) *box.Box {
	b := el(10, 10, 100, 40, func(s *style.Computed) {
		s.Background = white
		sh.Inset, sh.Color = true, shade
		s.Shadows = []style.Shadow{sh}
		if edit != nil {
			edit(s)
		}
	})
	return b
}

// covered is the area the fills of colour c draw, clipped as drawn.
func covered(r *recorder, c color.Color) float32 {
	var a float32
	for _, o := range r.of("rect") {
		if o.Color == c {
			x := max(o.Rect.X, o.Clip.X)
			y := max(o.Rect.Y, o.Clip.Y)
			x1 := min(o.Rect.X+o.Rect.W, o.Clip.X+o.Clip.W)
			y1 := min(o.Rect.Y+o.Rect.H, o.Clip.Y+o.Clip.H)
			if o.Clip == (geom.Rect{}) {
				x, y, x1, y1 = o.Rect.X, o.Rect.Y, o.Rect.X+o.Rect.W, o.Rect.Y+o.Rect.H
			}
			a += max(x1-x, 0) * max(y1-y, 0)
		}
	}
	return a
}

func TestAHardInsetShadowIsAFrameInsideThePaddingBox(t *testing.T) {
	r := pictures(insetBox(style.Shadow{Spread: 3}, nil))
	assert.InDelta(t, 100*40-94*34, covered(r, shade), 1e-3, "3 points all round, inside the box")
	for _, o := range r.of("rect") {
		if o.Color == shade {
			assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 40}, o.Clip, "clipped to the padding box")
		}
	}
	assert.Empty(t, r.of("gradient"), "no blur, no fade")
}

func TestAnOffsetInsetShadowShowsOnTheOffsetSide(t *testing.T) {
	r := pictures(insetBox(style.Shadow{X: 4, Y: 2}, nil))
	assert.InDelta(t, 100*40-96*38, covered(r, shade), 1e-3, "4 down the left, 2 along the top")
}

func TestAnInsetShadowIsDrawnOverTheBackgroundUnderTheBorder(t *testing.T) {
	b := insetBox(style.Shadow{Spread: 2}, func(s *style.Computed) {
		s.Border = style.All[float32](1)
		s.BorderStyle, s.BorderColor = style.All(style.BorderSolid), style.All(blue)
	})
	b.Border = box.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	r := pictures(b)
	var order []string
	for _, o := range r.ops {
		if o.Kind == "rect" {
			order = append(order, names[o.Color]+map[bool]string{true: "shade", false: ""}[o.Color == shade])
		}
	}
	require.NotEmpty(t, order)
	assert.Equal(t, "white", order[0])
	assert.Equal(t, "blue", order[len(order)-1])
	assert.InDelta(t, 98*38-94*34, covered(r, shade), 1e-3, "inside the border")
}

func TestABlurredInsetShadowFadesAcrossItsBlur(t *testing.T) {
	r := pictures(insetBox(style.Shadow{Blur: 8}, nil))
	fades := r.of("gradient")
	require.Len(t, fades, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 40}, fades[0].Rect, "a mesh over the padding box")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 40}, fades[0].Clip, "clipped to the box")
	assert.InDelta(t, 0.5*(1-blurredInterval(10, 10, 110, 4)*blurredInterval(10, 10, 50, 4)), fades[0].Color.A, 1e-4,
		"the corner: CSS's Gaussian, sigma half the blur")
	assert.Zero(t, covered(r, shade), "no hard bands when blurred")
}

func TestABlurredIntervalIsAHalfAtItsEdgesAndWholeInside(t *testing.T) {
	assert.InDelta(t, 0.5, blurredInterval(0, 0, 100, 4), 1e-4)
	assert.InDelta(t, 1, blurredInterval(50, 0, 100, 4), 1e-4)
	assert.InDelta(t, 0, blurredInterval(-20, 0, 100, 4), 1e-4)
}

func rounded(r float32) func(s *style.Computed) {
	return func(s *style.Computed) {
		c := style.Size[style.Length]{Width: style.Px(r), Height: style.Px(r)}
		s.Radius = style.Corners[style.Size[style.Length]]{TopLeft: c, TopRight: c, BottomRight: c, BottomLeft: c}
	}
}

// insideRounded reports whether p is inside the w×h box at (x, y) with
// corners of radius r, give or take tol.
func insideRounded(p geom.Vec2, x, y, w, h, r, tol float32) bool {
	q := geom.V2(geom.Abs(p.X-(x+w/2))-(w/2-r), geom.Abs(p.Y-(y+h/2))-(h/2-r))
	out := geom.V2(max(q.X, 0), max(q.Y, 0))
	return out.Len()+min(max(q.X, q.Y), 0)-r <= tol
}

func TestARoundedBoxsInsetShadowStaysInsideItsCorners(t *testing.T) {
	r := pictures(insetBox(style.Shadow{Spread: 3}, rounded(12)))
	meshes := r.of("gradient")
	require.Len(t, meshes, 1, "a mesh, clipped to the rounded box")
	for _, p := range meshes[0].Points {
		p = meshes[0].Xform.Xform(p)
		assert.True(t, insideRounded(p, 10, 10, 100, 40, 12, 0.01), "%v outside the rounding", p)
	}
	assert.Zero(t, covered(r, shade), "no square bands")
}

func TestARoundedInsetRingIsSolidAtTheEdgeAndClearInside(t *testing.T) {
	r := pictures(insetBox(style.Shadow{Spread: 3}, rounded(12)))
	m := r.of("gradient")[0]
	alphaAt := func(want geom.Vec2) float32 {
		best, at := float32(1e9), float32(-1)
		for i, p := range m.Points {
			if d := m.Xform.Xform(p).Sub(want).Len(); d < best {
				best, at = d, m.Alpha[i]
			}
		}
		return at
	}
	assert.InDelta(t, 0.5, alphaAt(geom.V2(60, 10)), 0.05, "the top edge's middle: the full shade")
	assert.InDelta(t, 0, alphaAt(geom.V2(60, 30)), 0.05, "the middle: clear")
	assert.InDelta(t, 0.5, alphaAt(geom.V2(14, 14)), 0.05, "inside the corner's curve, still the ring")
	assert.InDelta(t, 0, alphaAt(geom.V2(22, 22)), 0.05, "past the ring's rounded inner corner: clear")
}

func TestAMovedBoxKeepsItsInsetMesh(t *testing.T) {
	a := pictures(insetBox(style.Shadow{Blur: 8}, rounded(10))).of("gradient")[0]
	moved := insetBox(style.Shadow{Blur: 8}, rounded(10))
	moved.Pos = geom.V2(150, 90)
	b := pictures(moved).of("gradient")[0]
	assert.Equal(t, a.Points, b.Points, "the same mesh, in the box's own space")
	assert.Equal(t, geom.V2(150, 90), b.Xform.Origin, "placed by a translation")
}

func BenchmarkARoundedPanelsInsetShadows(b *testing.B) {
	panel := insetBox(style.Shadow{Blur: 12}, rounded(8))
	panel.Style.Shadows = append(panel.Style.Shadows, style.Shadow{Y: 1, Color: color.RGBA(1, 1, 1, 0.08), Inset: true})
	t := tree(el(0, 0, 400, 300, nil, panel))
	s := &probe{xf: geom.Identity2D()}
	b.ReportAllocs()
	for b.Loop() {
		Paint(s, t)
	}
}

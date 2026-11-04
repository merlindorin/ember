package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stops(cs ...any) []style.Stop {
	var out []style.Stop
	for _, c := range cs {
		switch v := c.(type) {
		case color.Color:
			out = append(out, style.Stop{Color: v})
		case style.Length:
			out[len(out)-1].At, out[len(out)-1].Placed = v, true
		}
	}
	return out
}

func pct(v float32) style.Length { return style.Length{Kind: style.Fixed, Percent: v, IsPercent: true} }
func pxl(v float32) style.Length { return style.Length{Kind: style.Fixed, Px: v} }

func TestStopsArePlacedAsCSSPlacesThem(t *testing.T) {
	at := func(g *style.Gradient, span float32) []float32 {
		var u []float32
		for _, s := range gradientSamples(g, span) {
			if len(u) == 0 || s.u != u[len(u)-1] {
				u = append(u, s.u)
			}
		}
		return u
	}
	g := &style.Gradient{Stops: stops(red, blue, white)}
	s := gradientSamples(g, 200)
	assert.Equal(t, float32(0), s[0].u, "the first at the start")
	assert.Equal(t, float32(200), s[len(s)-1].u, "the last at the end")
	assert.Contains(t, at(g, 200), float32(100), "the middle one halfway")

	g = &style.Gradient{Stops: stops(red, pct(50), blue, pxl(40), white)}
	s = gradientSamples(g, 200)
	assert.Equal(t, float32(100), s[0].u)
	assert.Equal(t, float32(100), s[1].u, "a stop placed before another stands with it")
	assert.Equal(t, float32(200), s[len(s)-1].u)

	g = &style.Gradient{Stops: stops(red, pxl(20), blue, white, black, pxl(80))}
	u := at(g, 200)
	assert.Contains(t, u, float32(40), "unplaced stops share the room between placed ones")
	assert.Contains(t, u, float32(60))
}

func TestAGradientBlendsInEncodedSRGBAsChromeDoes(t *testing.T) {
	g := &style.Gradient{Stops: stops(black, white)}
	s := gradientSamples(g, 120)
	mid := s[len(s)/2]
	require.InDelta(t, 60, mid.u, 0.01)
	e := decoded(mid.rgba).SRGB()
	assert.InDelta(t, 0.5, e[0], 0.01, "half way is encoded 0.5, as Chrome's, not linear light's 0.735")
}

func TestACornersLineIsSquareToTheOtherDiagonal(t *testing.T) {
	area := geom.Rect{W: 200, H: 100}
	d := lineDirection(&style.Gradient{Corner: [2]int8{1, -1}}, area)
	diagonal := geom.V2(200, 100)
	assert.InDelta(t, 0, d.Dot(diagonal), 1e-3, "the 50% line runs through the corners beside")
	assert.Positive(t, d.X)
	assert.Negative(t, d.Y)
	d = lineDirection(&style.Gradient{Angle: 90}, area)
	assert.InDelta(t, 1, d.X, 1e-6)
}

func TestAGradientFillsItsRoundedPaddingBoxAndNothingOutside(t *testing.T) {
	r := newRecorder()
	p := &painter{s: r, base: geom.Identity2D()}
	area := geom.Rect{X: 10, Y: 20, W: 100, H: 60}
	p.gradient(area, radii{12, 12, 12, 12}, &style.Gradient{Angle: 90, Stops: stops(red, blue)}, 1)
	require.Len(t, r.ops, 3)
	clip := r.ops[0]
	assert.Equal(t, "push-round", clip.Kind, "the box's curve clips the bands, anti-aliased")
	assert.Equal(t, area, clip.Rect)
	assert.Equal(t, [4]float32{12, 12, 12, 12}, clip.Radii)
	assert.Equal(t, "pop", r.ops[2].Kind)
	o := r.ops[1]
	assert.Equal(t, area, o.Rect, "the mesh spans the box")
	for _, q := range o.Points {
		assert.True(t, q.X >= area.X-0.01 && q.X <= area.X+area.W+0.01 && q.Y >= area.Y-0.01 && q.Y <= area.Y+area.H+0.01,
			"%v inside the box", q)
	}
	for i, q := range o.Points {
		switch {
		case q.X <= area.X+0.01:
			assert.InDeltaSlice(t, []float32{red.R, red.G, red.B, red.A},
				[]float32{o.Colors[i].R, o.Colors[i].G, o.Colors[i].B, o.Colors[i].A}, 1e-6, "the start is the first stop")
		case q.X >= area.X+area.W-0.01:
			assert.InDeltaSlice(t, []float32{blue.R, blue.G, blue.B, blue.A},
				[]float32{o.Colors[i].R, o.Colors[i].G, o.Colors[i].B, o.Colors[i].A}, 1e-6, "the end the last")
		}
	}
}

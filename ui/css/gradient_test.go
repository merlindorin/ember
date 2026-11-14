package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gradientOf(t *testing.T, decl string) *style.Gradient {
	t.Helper()
	d := el(dom.Div, "g")
	Compute(page(d), []*Sheet{sheet(t, "#g { "+decl+" }")})
	g := styleOf(d).BackgroundImage.Gradient
	require.NotNil(t, g, decl)
	return g
}

func TestALinearGradientReadsItsDirection(t *testing.T) {
	for decl, want := range map[string]float32{
		"background-image: linear-gradient(red, blue)":              180,
		"background-image: linear-gradient(to right, red, blue)":    90,
		"background-image: linear-gradient(to top, red, blue)":      0,
		"background-image: linear-gradient(to left, red, blue)":     270,
		"background-image: linear-gradient(45deg, red, blue)":       45,
		"background-image: linear-gradient(0.25turn, red, blue)":    90,
		"background: linear-gradient(to bottom, red, blue) #000":    180,
		"background-image: linear-gradient(-90deg, #fff, #000 50%)": -90,
		"background-image: LINEAR-GRADIENT(TO RIGHT, RED, BLUE)":    90,
	} {
		g := gradientOf(t, decl)
		assert.Equal(t, want, g.Angle, decl)
		assert.Zero(t, g.Corner, decl)
	}
	g := gradientOf(t, "background-image: linear-gradient(to bottom left, red, blue)")
	assert.Equal(t, [2]int8{-1, 1}, g.Corner, "a corner's angle waits for the box")
}

func TestALinearGradientsStopsKeepTheirPositions(t *testing.T) {
	g := gradientOf(t, "background-image: linear-gradient(red, #0f0 20px 40%, blue 90%, white)")
	require.Len(t, g.Stops, 5, "a stop with two positions is two")
	assert.Equal(t, hex("red"), g.Stops[0].Color)
	assert.False(t, g.Stops[0].Placed)
	assert.Equal(t, style.Stop{Color: hex("#0f0"), At: style.Length{Kind: style.Fixed, Px: 20}, Placed: true}, g.Stops[1])
	assert.Equal(t, float32(40), g.Stops[2].At.Percent)
	assert.Equal(t, hex("#0f0"), g.Stops[2].Color)
	assert.Equal(t, float32(90), g.Stops[3].At.Percent)
	assert.False(t, g.Stops[4].Placed)
}

func TestABadLinearGradientIsDropped(t *testing.T) {
	for _, src := range []string{
		"#g { background-image: linear-gradient(red) }",
		"#g { background-image: linear-gradient(to middle, red, blue) }",
		"#g { background-image: linear-gradient(red, 10%, blue) }",
		"#g { background-image: conic-gradient(red 10px, blue) }",
		"#g { background-image: radial-gradient(square, red, blue) }",
		"#g { background-image: radial-gradient(circle at, red, blue) }",
		"#g { background-image: repeating-linear-gradient(red, blue) }",
	} {
		_, warns := Parse("t.css", src)
		assert.NotEmpty(t, warns, src)
	}
}

func TestAConicGradientReadsItsStartCentreAndStopsAsATurn(t *testing.T) {
	g := gradientOf(t,
		"background-image: conic-gradient(from 90deg at 25% 75%, transparent 0 70%, rgba(0,0,0,.72) 70% 100%)")
	assert.Equal(t, style.ConicGradient, g.Kind)
	assert.Equal(t, float32(90), g.Angle)
	assert.Equal(t, float32(25), g.At.X.Percent)
	assert.Equal(t, float32(75), g.At.Y.Percent)
	require.Len(t, g.Stops, 4, "two stops with two positions each")
	assert.Equal(t, float32(0), g.Stops[0].At.Percent)
	assert.Equal(t, float32(70), g.Stops[1].At.Percent)
	assert.Equal(t, float32(70), g.Stops[2].At.Percent, "a hard stop")
	g = gradientOf(t, "background-image: conic-gradient(red 90deg, blue 0.5turn)")
	assert.Equal(t, float32(0), g.Angle, "from 0 by default")
	assert.Equal(t, float32(50), g.At.X.Percent, "at the middle by default")
	assert.InDelta(t, 25, g.Stops[0].At.Percent, 1e-4, "degrees as a share of a turn")
	assert.InDelta(t, 50, g.Stops[1].At.Percent, 1e-4)
}

func TestARadialGradientReadsItsShapeSizeAndCentre(t *testing.T) {
	g := gradientOf(t, "background-image: radial-gradient(circle at 50% 50%, transparent 30%, rgba(0,0,0,.55) 85%)")
	assert.Equal(t, style.RadialGradient, g.Kind)
	assert.True(t, g.Circle)
	assert.Equal(t, style.FarthestCorner, g.Size, "to the farthest corner by default")
	assert.Equal(t, float32(30), g.Stops[0].At.Percent)
	g = gradientOf(t, "background-image: radial-gradient(#3a2a1e 40%, #120c09)")
	assert.False(t, g.Circle, "an ellipse by default")
	assert.Equal(t, float32(50), g.At.Y.Percent)
	g = gradientOf(t, "background: radial-gradient(closest-side ellipse at 20px 30%, red, blue 12px)")
	assert.Equal(t, style.ClosestSide, g.Size)
	assert.Equal(t, float32(20), g.At.X.Px)
	assert.Equal(t, float32(12), g.Stops[1].At.Px)
}

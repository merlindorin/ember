package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shadowed(shadows ...style.Shadow) *recorder {
	p := el(0, 0, 100, 20, func(s *style.Computed) { s.Color = white; s.TextShadows = shadows })
	p.Append(text(0, 0, "hi", p.Style))
	r := newRecorder()
	Paint(r, tree(p))
	return r
}

func TestASharpTextShadowIsTheTextMovedInItsColourUnderneath(t *testing.T) {
	ts := shadowed(style.Shadow{X: 1, Y: 1.5, Color: color.RGBA(0, 0, 0, 0.8)}).of("text")
	require.Len(t, ts, 2)
	assert.Equal(t, geom.V2(1, 16.5), ts[0].At)
	assert.Equal(t, color.RGBA(0, 0, 0, 0.8), ts[0].Color)
	assert.Equal(t, geom.V2(0, 15), ts[1].At, "the text itself last, on top")
	assert.Equal(t, white, ts[1].Color)
}

func TestTheFirstTextShadowIsOnTopOfTheOthers(t *testing.T) {
	ts := shadowed(style.Shadow{X: 1, Color: red}, style.Shadow{X: 2, Color: blue}).of("text")
	require.Len(t, ts, 3)
	assert.Equal(t, []color.Color{blue, red, white}, []color.Color{ts[0].Color, ts[1].Color, ts[2].Color})
}

func TestACurrentColorTextShadowTakesTheTextsColour(t *testing.T) {
	ts := shadowed(style.Shadow{Y: 2, CurrentColor: true}).of("text")
	require.Len(t, ts, 2)
	assert.Equal(t, white, ts[0].Color)
}

func TestABlurredTextShadowSpreadsItsInkOverTaps(t *testing.T) {
	ts := shadowed(style.Shadow{X: 2, Y: 2, Blur: 4, Color: black}).of("text")
	require.Greater(t, len(ts), 2)
	taps := ts[:len(ts)-1]
	var sum float32
	lo, hi := taps[0].At, taps[0].At
	for _, o := range taps {
		sum += o.Color.A
		lo, hi = geom.V2(min(lo.X, o.At.X), min(lo.Y, o.At.Y)), geom.V2(max(hi.X, o.At.X), max(hi.Y, o.At.Y))
	}
	assert.InDelta(t, 1, sum, 0.01, "the ink of one shadow, shared out")
	assert.InDelta(t, 2, (lo.X+hi.X)/2, 1e-4, "centred on the offset")
	assert.InDelta(t, 17, (lo.Y+hi.Y)/2, 1e-4)
	assert.Greater(t, hi.X-lo.X, float32(0))
}

func TestATransparentTextShadowDrawsNothing(t *testing.T) {
	assert.Len(t, shadowed(style.Shadow{X: 1, Color: color.RGBA(0, 0, 0, 0)}).of("text"), 1)
}

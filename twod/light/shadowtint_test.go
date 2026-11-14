package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAShadowLetsThroughItsColourOverWhatOpacityLeaves(t *testing.T) {
	w, win, _ := scene(t)
	lamp := NewPoint(50, color.White, ShadowColor(color.RGB(0, 0.2, 0.6)), ShadowOpacity(0.5))
	layer := twod.NewLayer(0, Lit()).Add(lamp)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.Len(t, l.Lights, 1)
	tint := l.Lights[0].ShadowTint
	assert.InDelta(t, 0.5, tint.R, 1e-6)
	assert.InDelta(t, 0.6, tint.G, 1e-6)
	assert.InDelta(t, 0.8, tint.B, 1e-6)

	lamp.SetShadowOpacity(1)
	lamp.SetShadowColor(color.Black)
	w.RunFrames(1)
	assert.Equal(
		t,
		color.RGB(0, 0, 0),
		draw.LightingFor(w, win).Lights[0].ShadowTint,
		"black and whole, the default, lets nothing through",
	)
	assert.Equal(t, float32(1), lamp.ShadowOpacity())
	assert.Equal(t, color.Black, lamp.ShadowColor())
	assert.Equal(t, float32(1), NewPoint(10, color.White).ShadowOpacity())
	assert.Equal(t, float32(0), NewPoint(10, color.White, ShadowOpacity(-1)).ShadowOpacity())
}

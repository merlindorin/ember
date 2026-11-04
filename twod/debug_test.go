package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/stretchr/testify/assert"
)

type fixedLighting struct{ l *draw.Lighting }

func (f fixedLighting) Lighting(*platform.Window) *draw.Lighting { return f.l }

func TestF3ListsTheLightingAndF5AndF6SwitchItsDebugViews(t *testing.T) {
	w, win, _ := app()
	draw.InstallLighting(w, fixedLighting{&draw.Lighting{
		Lights:    []draw.Light{{}, {}, {XRay: true}},
		Occluders: &draw.List{Vertices: make([]draw.Vertex, 8)},
		Static:    &draw.StaticLights{Count: 4},
	}})
	in := wm.Input(w)
	tap := func(k input.Key) {
		in.SetKey(k, true)
		w.RunFrames(1)
		in.SetKey(k, false)
		w.RunFrames(1)
	}
	tap(input.KeyF5)
	assert.Equal(t, draw.LightViewOff, draw.LightDebugOf(w, win).View, "F5 does nothing while F3 is hidden")

	tap(input.KeyF3)
	assert.Equal(t, []string{
		"lights 3 (2 cast) of 256, 4 static  occluders 2",
		"F5 view: off  F6 outlines: off",
	}, debugLights(w))
	var seen []draw.LightView
	for range 4 {
		tap(input.KeyF5)
		seen = append(seen, draw.LightDebugOf(w, win).View)
	}
	assert.Equal(
		t,
		[]draw.LightView{draw.LightViewPicture, draw.LightViewOccluders, draw.LightViewShadows, draw.LightViewOff},
		seen,
	)

	tap(input.KeyF5)
	assert.NotZero(t, draw.DebugPanel(w).W, "the panel's place, left clear of the view")
	tap(input.KeyF6)
	assert.True(t, draw.LightDebugOf(w, win).Outlines)
	assert.Equal(t, "F5 view: light picture  F6 outlines: on", debugLights(w)[1])
	tap(input.KeyF6)
	assert.False(t, draw.LightDebugOf(w, win).Outlines)
	tap(input.KeyF3)
	assert.Zero(t, draw.DebugPanel(w), "hidden, it leaves no place clear")
}

func TestF3ListsNoLightingForAWorldWithout(t *testing.T) {
	w, _, _ := app()
	assert.Empty(t, debugLights(w))
}

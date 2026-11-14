package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestALayerALightAffectsDrawsInALightClassOfItsOwn(t *testing.T) {
	w, win, _ := scene(t)
	inner := twod.NewLayer(2, Lit()).Add(twod.NewSprite(picture(4, 4), twod.At(20, 0)))
	floor := twod.NewLayer(0, Lit()).Add(twod.NewSprite(picture(4, 4)), inner)
	other := twod.NewLayer(1, Lit()).Add(twod.NewSprite(picture(4, 4), twod.At(-20, 0)))
	lamp := NewPoint(50, color.White, Affects(floor))
	win.Add(floor, other, lamp)
	w.AddEntities(floor, other, lamp)
	w.RunFrames(2)
	assert.Equal(t, []*twod.Layer{floor}, lamp.Affects())
	classes := draw.ClassesFor(w, win)
	require.Len(t, classes, 1, "the floor alone is named")
	assert.Equal(t, []any{floor}, classes[0])
	byX := map[float32]int{}
	for i, v := range draw.FrontFor(w, win).Vertices {
		if i%4 == 0 {
			byX[v.Pos.X] = v.Class()
		}
	}
	assert.Equal(
		t,
		map[float32]int{396: 1, 436: 1, 356: 0},
		byX,
		"the floor and the layer in it are of its class, the other layer of none",
	)

	lamp.SetAffects(floor, inner)
	w.RunFrames(1)
	classes = draw.ClassesFor(w, win)
	require.Len(t, classes, 2)
	assert.Equal(t, []any{inner, floor}, classes[1], "a named layer in a named layer: both, the innermost first")
	l := draw.LightingFor(w, win)
	assert.Len(t, l.Plan.Pictures, 1, "only the other layer is left out of the lamp")
}

func TestFallingBackPastTheCapIsToldOnceAWindow(t *testing.T) {
	var out bytes.Buffer
	log.SetOutput(&out)
	defer log.SetOutput(os.Stderr)
	w, win, _ := scene(t)
	for i := range 6 {
		layer := twod.NewLayer(i, Lit()).Add(twod.NewSprite(picture(4, 4), twod.At(float32(10*i), 0)))
		lamp := NewPoint(50, color.White, Affects(layer))
		win.Add(layer, lamp)
		w.AddEntities(layer, lamp)
	}
	w.RunFrames(3)
	l := draw.LightingFor(w, win)
	assert.Len(t, l.Plan.Pictures, draw.MaxPictures)
	assert.NotEmpty(t, l.Plan.Fallen)
	assert.Equal(t, 1, strings.Count(out.String(), "needs more than 4 light pictures"), out.String())
	assert.Contains(t, out.String(), "layer layer")
}

func TestExposureIsTheCamerasAndOneByDefault(t *testing.T) {
	w, win, cam := scene(t)
	layer := twod.NewLayer(0, Lit()).Add(NewPoint(10, color.White))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(1)
	assert.Equal(t, float32(1), ExposureOf(cam))
	assert.Equal(t, float32(1), draw.LightingFor(w, win).Exposure)
	SetExposure(cam, 1.3)
	w.RunFrames(1)
	assert.Equal(t, float32(1.3), ExposureOf(cam))
	assert.Equal(t, float32(1.3), draw.LightingFor(w, win).Exposure)
	SetExposure(cam, -1)
	assert.Equal(t, float32(0.01), ExposureOf(cam), "never below 0.01")
}

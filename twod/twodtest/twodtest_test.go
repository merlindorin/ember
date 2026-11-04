package twodtest_test

import (
	"image"
	"slices"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/twod/twodtest"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scene is an 800×600 window whose camera looks at the world's origin, so
// a world point p shows at (400, 300) + p.
func scene(t *testing.T, opts ...twod.Option) (*world.World, *platform.Window) {
	t.Helper()
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(800, 600))
	cam := twod.NewCamera()
	for _, o := range opts {
		o(cam)
	}
	win.Add(cam)
	w.AddEntities(win)
	return w, win
}

func picture(w, h int) *twod.Texture {
	return twod.NewTexture(image.NewRGBA(image.Rect(0, 0, w, h)))
}

func TestAnEntityIsDrawnWhereItShowsInTheWindow(t *testing.T) {
	w, win := scene(t)
	crate := twod.NewSprite(picture(16, 16), twod.At(-100, 50))
	win.Add(crate)
	w.AddEntities(crate)
	w.RunFrames(1)
	at, ok := twodtest.Drawn(w, crate)
	require.True(t, ok)
	assert.Equal(t, geom.V2(300, 350), at.Origin, "its position, in the window's points")
	_, ok = twodtest.Drawn(world.New(), crate)
	assert.False(t, ok, "a World drawing no 2D")
}

func TestAPassesPictureIsFoundAndShownByTheWindow(t *testing.T) {
	w, win := scene(t)
	minimap := twod.NewRenderTexture(64, 64)
	hud := twod.NewLayer(1, twod.Fixed()).Add(twod.NewSprite(minimap, twod.At(700, 60)))
	win.Add(twod.NewSprite(picture(8, 8)), twod.NewCamera(twod.Into(minimap), twod.Name("minimap-camera")), hud)
	w.AddEntities(hud)
	w.RunFrames(1)
	passes := twodtest.Passes(w, win)
	i := slices.IndexFunc(passes, func(p twodtest.Pass) bool { return p.Name == "minimap-camera" })
	require.GreaterOrEqual(t, i, 0, "the camera's pass, by its name: %+v", passes)
	assert.Same(t, &minimap.Texture, passes[i].Target)
	assert.Contains(t, twodtest.Shown(w, win), passes[i].Target, "the HUD shows its picture")
}

func TestTheLightIsReadInTheWindowsPointsWithItsOccluders(t *testing.T) {
	w, win := scene(t, light.Ambient(color.RGB(0.1, 0.1, 0.2)))
	lamp := light.NewPoint(120, color.White, light.At(100, 0))
	block := twod.NewSprite(picture(20, 20), twod.At(-100, 0), light.Occluder(2))
	layer := twod.NewLayer(0, light.Lit()).Add(lamp, block)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l, ok := twodtest.LightingOf(w, win)
	require.True(t, ok)
	assert.Equal(t, color.RGB(0.1, 0.1, 0.2), l.Ambient)
	require.Len(t, l.Lights, 1)
	assert.InDelta(t, 500, l.Lights[0].At.X, 0.01, "the lamp in the window's points")
	assert.InDelta(t, 300, l.Lights[0].At.Y, 0.01)
	assert.Equal(t, float32(120), l.Lights[0].Radius)
	require.NotEmpty(t, l.Occluders)
	assert.Zero(t, len(l.Occluders)%4, "four vertices a box")
	for _, v := range l.Occluders {
		assert.Equal(t, uint8(1<<2), v.Groups, "its group, as a bit")
		assert.InDelta(
			t,
			300,
			v.At.X,
			11,
			"round the block (20 points and half a point of padding), in the window's points",
		)
		assert.InDelta(t, 300, v.At.Y, 11)
	}
	_, ok = twodtest.LightingOf(world.New(), platform.NewWindow())
	assert.False(t, ok, "nothing lights a window of a world without light")
}

func TestAnsweredProbesReadTheLightGiven(t *testing.T) {
	w, win := scene(t, light.Ambient(color.Black))
	probe := light.NewProbe(twod.At(10, 10))
	layer := twod.NewLayer(0, light.Lit()).Add(probe)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	require.False(t, probe.Ready(), "without a GPU, no answer")
	twodtest.AnswerProbes(w, win, color.Gray(0.4))
	assert.True(t, probe.Ready())
	assert.Equal(t, color.Gray(0.4), probe.Light())
}

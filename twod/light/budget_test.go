package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"math"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// brightness is the red of l's light at x on the row every budget test puts
// its lights (y 300, in window points), 0 when it draws none there.
func brightness(l *draw.Lighting, x float32) float32 {
	for _, li := range l.Lights {
		if li.At.X == x && li.At.Y == 300 {
			return li.Color.R
		}
	}
	return 0
}

func TestABudgetKeepsTheLightsNearestTheMiddle(t *testing.T) {
	w, win, cam := scene(t, Budget(2), BudgetFade(0))
	layer := twod.NewLayer(0, Lit()).Add(
		NewPoint(10, color.White, At(150, 0)),
		NewPoint(10, color.White, At(10, 0)),
		NewPoint(10, color.White, At(-100, 50)),
		NewPoint(10, color.White, At(-30, 0)),
		NewPoint(10, color.White, At(0, 0), Shadows(false), Static()),
		NewDirectional(color.White, 0),
	)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.Len(t, l.Lights, 3, "the two nearest and the sun, which counts against none")
	assert.Equal(t, float32(1), brightness(l, 420))
	assert.Equal(t, float32(1), brightness(l, 340))
	assert.Equal(t, 2, l.Budget)
	assert.Equal(t, 2, BudgetOf(cam))

	SetBudget(cam, 0)
	w.RunFrames(1)
	assert.Len(t, draw.LightingFor(w, win).Lights, 5, "no budget draws every light, the static one baked")
}

func TestPriorityComesBeforeDistance(t *testing.T) {
	w, win, _ := scene(t, Budget(1), BudgetFade(0))
	far := NewPoint(10, color.White, At(150, 0), Priority(1))
	layer := twod.NewLayer(0, Lit()).Add(NewPoint(10, color.White, At(0, 0)), far)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.Len(t, l.Lights, 1)
	assert.Equal(t, float32(1), brightness(l, 700))

	far.SetPriority(0)
	w.RunFrames(1)
	assert.Equal(t, float32(1), brightness(draw.LightingFor(w, win), 400))
	assert.Equal(t, 0, far.Priority())
}

func TestALightFallingPastTheBudgetFadesOutThenBackIn(t *testing.T) {
	w, win, cam := scene(t, Budget(1), BudgetFade(0.5))
	near := NewPoint(10, color.White, At(0, 0))
	far := NewPoint(10, color.White, At(100, 0))
	layer := twod.NewLayer(0, Lit()).Add(far)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	require.Equal(t, float32(1), brightness(draw.LightingFor(w, win), 600), "alone, it is in at once")

	layer.Add(near)
	w.RunFrames(15)
	l := draw.LightingFor(w, win)
	assert.Equal(t, float32(1), brightness(l, 400), "the nearer one comes in at once, as it first shows")
	assert.InDelta(t, 0.5, brightness(l, 600), 0.1, "half way through its fade")
	assert.Equal(t, 1, l.Fading)
	w.RunFrames(16)
	l = draw.LightingFor(w, win)
	assert.Equal(t, float32(0), brightness(l, 600), "out")
	assert.Len(t, l.Lights, 1)

	SetBudget(cam, 2)
	w.RunFrames(1)
	b := brightness(draw.LightingFor(w, win), 600)
	assert.True(t, b > 0 && b < 0.1, "back, fading in from nothing: %v", b)
	w.RunFrames(30)
	assert.Equal(t, float32(1), brightness(draw.LightingFor(w, win), 600))
	assert.Equal(t, float32(0.5), BudgetFadeOf(cam))
}

func TestAKeptLightStaysForOneBarelyNearer(t *testing.T) {
	w, win, _ := scene(t, Budget(1), BudgetFade(0))
	first := NewPoint(10, color.White, At(50, 0))
	layer := twod.NewLayer(0, Lit()).Add(first)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	layer.Add(NewPoint(10, color.White, At(-48, 0)))
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	assert.Equal(t, float32(1), brightness(l, 500), "kept: the other is only a little nearer")
	assert.Equal(t, float32(0), brightness(l, 304))
}

func TestTwoEqualLightsCrossingAtTheBudgetSwitchAtMostOnce(t *testing.T) {
	w, win, _ := scene(t, Budget(1), BudgetFade(0))
	still := NewPoint(10, color.White, At(50, 0))
	moving := NewPoint(10, color.White, At(-53.5, 0))
	layer := twod.NewLayer(0, Lit()).Add(still, moving)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(1)
	keeping := func() *Light {
		l := draw.LightingFor(w, win)
		require.Len(t, l.Lights, 1)
		if l.Lights[0].At.X > 400 {
			return still
		}
		return moving
	}
	last, switches := keeping(), 0
	for i := range 120 {
		x := float32(-50 + 3.5*math.Sin(float64(i)/6))
		moving.SetPosition(geom.V2(x, 0))
		w.RunFrames(1)
		if k := keeping(); k != last {
			last, switches = k, switches+1
		}
	}
	assert.LessOrEqual(t, switches, 1, "within a tenth of each other, the one kept stays")
}

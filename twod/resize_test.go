package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shown is how much of the world cam shows across its window, w, h in
// points, from the last frame.
func shown(cam *Camera, w, h float32) geom.Vec2 {
	return cam.ScreenToWorld(geom.V2(w, h)).Sub(cam.ScreenToWorld(geom.Vec2{}))
}

func TestABiggerWindowShowsMoreWorldByDefault(t *testing.T) {
	w, win, _ := app()
	cam := NewCamera()
	win.Add(cam)
	w.AddEntities(cam)
	w.RunFrames(1)
	assert.Equal(t, geom.V2(800, 600), shown(cam, 800, 600))
	win.SetSize(geom.V2(1600, 900))
	w.RunFrames(1)
	assert.Equal(t, geom.V2(1600, 900), shown(cam, 1600, 900), "the same scale, more world")
}

func TestViewSizeKeepsTheWorldInViewAndScales(t *testing.T) {
	w, win, _ := app()
	tall := NewCamera(ViewSize(0, 300))
	win.Add(tall)
	w.AddEntities(tall)
	w.RunFrames(1)
	assert.Equal(t, geom.V2(400, 300), shown(tall, 800, 600), "300 high, scaled by 2")
	win.SetSize(geom.V2(1600, 600))
	w.RunFrames(1)
	assert.Equal(t, geom.V2(800, 300), shown(tall, 1600, 600), "still 300 high, wider with the window")
	win.SetSize(geom.V2(800, 1200))
	w.RunFrames(1)
	assert.Equal(t, geom.V2(200, 300), shown(tall, 800, 1200))
	x, y := tall.ViewSize()
	assert.Equal(t, [2]float32{0, 300}, [2]float32{x, y})

	both := NewCamera(ViewSize(400, 300), Zoom(2))
	both.SetPriority(1)
	win.Add(both)
	w.AddEntities(both)
	win.SetSize(geom.V2(1600, 600))
	w.RunFrames(1)
	assert.Equal(t, geom.V2(400, 150), shown(both, 1600, 600), "the tighter axis fits, then Zoom")
	both.SetViewSize(0, 0)
	w.RunFrames(1)
	assert.Equal(t, geom.V2(800, 300), shown(both, 1600, 600), "0, 0 frees both axes")
}

func TestAResolutionCameraScalesByWholeFactorsAsTheWindowResizes(t *testing.T) {
	w, win, c := app()
	cam := NewCamera(Resolution(320, 180), ViewSize(100, 100))
	win.Add(cam, NewSprite(atlas(16, 16)))
	w.AddEntities(cam)
	for _, step := range []struct {
		window geom.Vec2
		factor float32
	}{{geom.V2(800, 600), 2}, {geom.V2(1280, 720), 4}, {geom.V2(1279, 720), 3}} {
		win.Sync(step.window, step.window, true)
		w.RunFrames(1)
		require.NotEmpty(t, c.Unindexed())
		lo, hi := corners(c.Unindexed()[:6])
		assert.Equal(
			t,
			geom.V2(320, 180).Scale(step.factor),
			hi.Sub(lo),
			"%v: ViewSize is ignored, whole factors",
			step.window,
		)
	}
}

func TestAnchoredFixedLayersKeepTheirEdgeAsTheWindowResizes(t *testing.T) {
	w, win, c := app()
	hud := NewLayer(1, Fixed(), Anchor(1, 1))
	badge := NewSprite(atlas(16, 16), Pivot(1, 1), At(-10, -10))
	hud.Add(badge)
	win.Add(NewCamera(ViewSize(0, 100), At(5000, 0)), hud)
	w.AddEntities(hud)
	w.RunFrames(1)
	_, hi := corners(c.Unindexed())
	assert.Equal(t, geom.V2(790, 590), hi, "10 pt from the bottom-right corner")
	win.SetSize(geom.V2(1000, 700))
	w.RunFrames(1)
	_, hi = corners(c.Unindexed())
	assert.Equal(t, geom.V2(990, 690), hi, "the same frame the window grew")
	at, ok := toScreen(w, badge)
	require.True(t, ok)
	assert.Equal(t, geom.V2(990, 690), at)
	x, y := hud.Anchor()
	assert.Equal(t, [2]float32{1, 1}, [2]float32{x, y})
	hud.SetAnchor(0, 0)
	w.RunFrames(1)
	_, hi = corners(c.Unindexed())
	assert.Equal(t, geom.V2(-10, -10), hi)
	assert.Empty(t, draw.PassesFor(w, win))
}

func TestAnOffsetMovesAStillSceneTooWithTimePaused(t *testing.T) {
	w, win, list := app()
	crate := NewSprite(atlas(8, 8), At(100, 100))
	cam := NewCamera()
	win.Add(crate, cam)
	w.Time().Scale = 0
	w.RunFrames(3)
	before := drawnX(list)
	cam.SetOffset(40, 0)
	w.RunFrames(1)
	if moved := drawnX(list) - before; geom.Abs(moved+40) > 1e-3 {
		t.Fatalf("an offset of 40 moves the drawn scene 40 the other way, nothing else moving: %v", moved)
	}
	cam.SetOffset(0, 0)
	w.RunFrames(1)
	if back := drawnX(list); geom.Abs(back-before) > 1e-3 {
		t.Fatalf("and back: %v, was %v", back, before)
	}
}

func TestOffsetNudgesTheViewWithoutTouchingTheFollow(t *testing.T) {
	w, win, _ := app()
	hero := NewSprite(nil, At(100, 100))
	shaken := NewCamera(Follow(hero), Smoothing(4), Zoom(2))
	still := NewCamera(Follow(hero), Smoothing(4), Zoom(2))
	other := platform.NewWindow(platform.Size(800, 600))
	win.Add(hero, shaken)
	other.Add(still)
	w.AddEntities(hero, shaken, other)
	w.RunFrames(1)
	hero.SetPosition(geom.V2(300, 100))
	for i := range 10 {
		shaken.SetOffset(float32(i%2*2-1)*5, 3)
		w.RunFrames(1)
		assert.Equal(t, still.at, shaken.at, "frame %d: the shake never feeds the follow", i)
		assert.Equal(t, still.at.Add(geom.V2(float32(i%2*2-1)*5, 3)), shaken.Center())
		on := shaken.WorldToScreen(hero.Position()).Sub(still.WorldToScreen(hero.Position()))
		assert.Equal(t, geom.V2(float32(i%2*2-1)*-10, -6), on, "the world moves the other way, by the zoom")
	}
	shaken.SetOffset(0, 0)
	w.RunFrames(1)
	assert.Equal(t, still.Center(), shaken.Center())
	x, y := NewCamera(Offset(2, 3)).Offset()
	assert.Equal(t, [2]float32{2, 3}, [2]float32{x, y})
}

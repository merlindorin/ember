package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/internal/draw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quads(l *draw.List) int { return len(l.Unindexed()) / 6 }

func TestWhatNoCameraShowsIsNotDrawn(t *testing.T) {
	w, win, c := app()
	crowd := NewLayer(0, SortByY())
	for i := range 100 {
		crowd.Add(NewSprite(atlas(10, 10), At(float32(i)*80, 300)))
	}
	hud := NewLayer(5, Fixed()).Add(NewSprite(atlas(10, 10), At(5000, 5000)))
	win.Add(crowd, hud, NewCamera(At(400, 300)))
	w.AddEntities(crowd, hud)
	w.RunFrames(1)
	assert.Equal(
		t,
		11+1,
		quads(c),
		"the eleven sprites from x 0 to 800 in an 800-point view, and the fixed layer however far",
	)
}

func TestASpriteAtTheEdgeOrJustPastItIsKept(t *testing.T) {
	w, win, c := app()
	layer := NewLayer(0).Add(
		NewSprite(atlas(20, 20), At(-5, 300)),
		NewSprite(atlas(20, 20), At(800+10+cullMargin-1, 300)),
		NewSprite(atlas(20, 20), At(800+10+cullMargin+2, 300)),
		NewSprite(atlas(40, 4), At(400, 600+cullMargin+15), Rotation(1.5707964)),
	)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(1)
	assert.Equal(t, 3, quads(c), "half in, within the margin and turned so it reaches in; not the one past the margin")
}

func TestAMinimapCameraStillDrawsItsMarkers(t *testing.T) {
	w, win, c := app()
	world := NewLayer(0).Add(NewSprite(atlas(10, 10), At(400, 300)))
	marker := NewSprite(atlas(4, 4), At(3000, 0))
	markers := NewLayer(1).Add(marker)
	minimap := NewRenderTexture(100, 100)
	win.Add(
		world,
		markers,
		NewCamera(At(400, 300), Layers(world)),
		NewCamera(Into(minimap), At(3000, 0), Layers(markers)),
	)
	w.AddEntities(world, markers)
	w.RunFrames(1)
	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 1)
	assert.Equal(t, 1, quads(passes[0].List), "far from the main view but in the minimap's, so kept for it")
	assert.Equal(t, 1, quads(c), "and the main view draws only the world")
}

func TestOnlyTheVisibleAreSortedAndKeepTheirOrder(t *testing.T) {
	w, win, c := app()
	crowd := NewLayer(0, SortByY())
	ys := []float32{400, 5000, 100, -5000, 250}
	for i, y := range ys {
		crowd.Add(NewSprite(atlas(10, 10), At(100+float32(i)*100, y)))
	}
	win.Add(crowd, NewCamera(At(400, 300)))
	w.AddEntities(crowd)
	w.RunFrames(1)
	require.Equal(t, 3, quads(c))
	var got []float32
	for i := 0; i < len(c.Unindexed()); i += 6 {
		lo, hi := corners(c.Unindexed()[i : i+6])
		got = append(got, (lo.Y+hi.Y)/2)
	}
	assert.Equal(t, []float32{100, 250, 400}, got, "by y, the far two left out")
}

// unbounded draws without telling its bounds, counting its draws.
type unbounded struct {
	Item
	drawn int
}

func (u *unbounded) Draw(*Canvas) { u.drawn++ }

func TestEntitiesWithoutBoundsAreAlwaysDrawn(t *testing.T) {
	w, win, _ := app()
	far := Build(&unbounded{Item: Base()}, "far", At(9000, 9000))
	win.Add(far, NewCamera())
	w.AddEntities(far)
	w.RunFrames(1)
	assert.Equal(t, 1, far.drawn, "a custom Draw that reports no bounds is kept")
}

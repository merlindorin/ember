package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASilhouetteHandsItsPictureToTheGPUInsteadOfBlockingWithItsBox(t *testing.T) {
	w, win, _ := scene(t)
	hero := twod.NewSprite(picture(10, 20), twod.At(0, 0), Silhouette())
	lamp := NewPoint(200, color.White, At(-50, 10))
	layer := twod.NewLayer(0, Lit()).Add(hero)
	win.Add(layer, lamp)
	w.AddEntities(layer, lamp)
	w.RunFrames(2)
	li := draw.LightingFor(w, win)
	require.NotNil(t, li)
	assert.Empty(t, li.Occluders.Vertices, "no box blocks light")
	assert.True(t, SilhouetteOf(hero))
	require.Len(t, li.Silhouettes, 1)
	s := li.Silhouettes[0]
	assert.Equal(t, 0, s.Light)
	assert.Equal(t, uint8(1), s.Groups)
	require.Len(t, li.SilhouetteQuads, 1)
	q := li.SilhouetteQuads[0]
	r := draw.LightRect(win.Size())
	assert.Equal(t, geom.V2(400+0*2, 300+10*2).Sub(geom.V2(r.X, r.Y)), q.Foot, "where it stands, zoomed twice")
	assert.Equal(t, float32(2), q.Across)
	assert.Equal(t, float32(40), q.Height, "how tall it stands, zoomed twice")
	assert.Equal(t, hero.Rect(), q.Rect)

	lit := geom.Rect{X: 400 - 100 - 400 - r.X, Y: 300 + 20 - 400 - r.Y, W: 800, H: 800}
	assert.LessOrEqual(t, s.Area.W, lit.W, "its area is the lamp's reach")
	assert.True(
		t,
		s.Area.X >= 0 && s.Area.Y >= 0 && s.Area.X+s.Area.W <= r.W && s.Area.Y+s.Area.H <= r.H,
		"within the light rectangle: %v",
		s.Area,
	)
	assert.Equal(t, s.Area.W, s.Tile.W, "a tile as large as its area")
	assert.Equal(t, s.Area.H, s.Tile.H)
}

func TestSilhouetteTilesNeverOverlapAndOverflowOntoANewPage(t *testing.T) {
	side := draw.SilhouetteSide
	draw.SilhouetteSide = 60
	defer func() { draw.SilhouetteSide = side }()
	w, win, _ := scene(t)
	layer := twod.NewLayer(0, Lit())
	for i := range 4 {
		x := float32(-150 + i*100)
		layer.Add(twod.NewSprite(picture(10, 20), twod.At(x, 0), Silhouette()))
		win.Add(NewPoint(45, color.White, At(x-30, 10)))
	}
	win.Add(layer)
	w.AddEntities(win)
	w.RunFrames(2)
	li := draw.LightingFor(w, win)
	require.Len(t, li.Silhouettes, 4)
	pages := map[int]bool{}
	for i, a := range li.Silhouettes {
		pages[a.Page] = true
		assert.LessOrEqual(t, a.Tile.X+a.Tile.W, li.SilhouetteSide)
		assert.LessOrEqual(t, a.Tile.Y+a.Tile.H, li.SilhouetteSide)
		for _, b := range li.Silhouettes[i+1:] {
			overlap := a.Tile.X < b.Tile.X+b.Tile.W && b.Tile.X < a.Tile.X+a.Tile.W && a.Tile.Y < b.Tile.Y+b.Tile.H &&
				b.Tile.Y < a.Tile.Y+a.Tile.H
			assert.False(t, a.Page == b.Page && overlap, "tiles %v and %v overlap", a, b)
		}
	}
	assert.Greater(t, len(pages), 1, "a small page overflows onto another")
	assert.Len(t, li.SilhouetteQuads, 4, "a quad a sprite and casting light")
}

func TestASilhouetteIsLongerUnderALowLightAndCastByTheStrongest(t *testing.T) {
	foot := geom.V2(100, 100)
	low := draw.Light{At: geom.V2(0, 100), Radius: 400, Elevation: 60, Color: color.White, Groups: 1}
	tx, ok := projection(
		low,
		foot,
		geom.Rect{X: -5, Y: -20, W: 10, H: 20},
		40,
		geom.Transform2D{X: geom.V2(2, 0), Y: geom.V2(0, 2)},
	)
	require.True(t, ok)
	end := tx.Xform(geom.V2(0, -20))
	assert.InDelta(t, 100+100*40/(60-40), end.X, 0.01, "d·h/(e−h) long")

	lights := []draw.Light{
		{At: geom.V2(90, 100), Radius: 50, Color: color.White, Groups: 1},
		{At: geom.V2(0, 100), Radius: 400, Color: color.RGB(0.1, 0.1, 0.1), Groups: 1},
		{At: geom.V2(110, 100), Radius: 50, Color: color.White, Groups: 1},
		{At: geom.V2(100, 80), Radius: 50, Color: color.White, Groups: 1},
		{At: geom.V2(100, 120), Radius: 50, Color: color.White, Groups: 2},
		{Kind: draw.DirectionalLight, Dir: geom.V2(0, 1), Length: 30, Color: color.White, Groups: 1},
		{At: geom.V2(100, 110), Radius: 50, Color: color.White, Groups: 1, XRay: true},
		{At: geom.V2(500, 500), Radius: 50, Color: color.White, Groups: 1},
	}
	var nears []near
	assert.ElementsMatch(t, []int{5, 0, 2, 3}, casting(lights, foot, 1, nil, &nears),
		"the sun, then the three strongest reaching it of its group, with shadows")
}

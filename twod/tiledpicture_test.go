package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type painted struct {
	tile [2]int
	area image.Rectangle
	cell [2]int
}

// pictureIn is a world with a 100×100 window, its camera at at, showing
// pic, and the calls pic's paint got.
func pictureIn(
	t *testing.T,
	at geom.Vec2,
	opts ...TiledPictureOption,
) (*world.World, *platform.Window, *Camera, *TiledPicture, *[]painted) {
	t.Helper()
	var calls []painted
	pic := NewTiledPicture(func(pt PictureTile) {
		cx, cy := pt.Cell(pt.Area.Min.X, pt.Area.Min.Y)
		calls = append(calls, painted{[2]int{pt.X, pt.Y}, pt.Area, [2]int{cx, cy}})
	}, opts...)
	w := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	win := platform.NewWindow(platform.Size(100, 100))
	cam := NewCamera(At(at.X, at.Y))
	win.Add(cam, NewLayer(0).Add(pic))
	w.AddEntities(win)
	w.RunFrames(1)
	return w, win, cam, pic, &calls
}

func TestATiledPicturePaintsTheTilesInViewWholeWithTheirBorder(t *testing.T) {
	_, _, _, pic, calls := pictureIn(t, geom.V2(50, 50), TexelSize(10, 10), TileTexels(8), PaintAhead(0))
	require.Len(t, *calls, 4, "the view, 0 to 100 points, is 2×2 tiles of 8 texels of 10 points")
	got := map[[2]int]painted{}
	for _, c := range *calls {
		got[c.tile] = c
	}
	one := got[[2]int{1, 1}]
	assert.Equal(t, 10, one.area.Dx(), "8 texels and a border each side")
	assert.Equal(
		t,
		[2]int{8 - 1, 8 - 1},
		one.cell,
		"its first texel to paint is the border, the world texel before tile 1's first",
	)
	assert.Equal(t, 1, pic.Pages())
	assert.Equal(t, 4, pic.Tiles())
}

func TestMarkDirtyRepaintsOnlyItsTexelsAndTheBordersThatShowThem(t *testing.T) {
	w, _, _, pic, calls := pictureIn(t, geom.V2(50, 50), TexelSize(10, 10), TileTexels(8), PaintAhead(0))
	*calls = (*calls)[:0]
	pic.MarkDirty(geom.Rect{X: 31, Y: 31, W: 2, H: 2})
	w.RunFrames(1)
	require.Len(t, *calls, 1, "a texel in the middle of tile 0, 0")
	assert.Equal(t, 1, (*calls)[0].area.Dx())
	assert.Equal(t, [2]int{3, 3}, (*calls)[0].cell)
	*calls = (*calls)[:0]
	pic.MarkDirty(geom.Rect{X: 71, Y: 31, W: 2, H: 2})
	w.RunFrames(1)
	tiles := map[[2]int]bool{}
	for _, c := range *calls {
		tiles[c.tile] = true
		assert.Equal(t, [2]int{7, 3}, c.cell, "the same world texel, in each tile that shows it")
	}
	assert.Equal(
		t,
		map[[2]int]bool{{0, 0}: true, {1, 0}: true},
		tiles,
		"texel 7 is tile 0's last and tile 1's left border",
	)
	*calls = (*calls)[:0]
	w.RunFrames(1)
	assert.Empty(t, *calls, "nothing dirty, nothing painted")
}

func TestOutOfViewTilesPastKeepTilesGiveTheirPlaceUp(t *testing.T) {
	w, _, cam, pic, calls := pictureIn(t, geom.V2(500, 500), TileTexels(1022), KeepTiles(1), PaintAhead(0))
	for i := 1; i <= 6; i++ {
		cam.SetPosition(geom.V2(500+float32(i)*1022, 500))
		w.RunFrames(1)
	}
	assert.Equal(t, 1, pic.Pages(), "a page holds 4 tiles of 1,024: the view needs 1, KeepTiles 1 more")
	assert.LessOrEqual(t, pic.Tiles(), 4)
	n := len(*calls)
	cam.SetPosition(geom.V2(500, 500))
	w.RunFrames(1)
	assert.Equal(t, n+1, len(*calls), "the first tile was given up: it paints again as it comes back")
}

func TestATileLeftOfTheOriginCountsBackwards(t *testing.T) {
	_, _, _, _, calls := pictureIn(t, geom.V2(-40, -40), TexelSize(10, 10), TileTexels(8), PaintAhead(0))
	got := map[[2]int]painted{}
	for _, c := range *calls {
		got[c.tile] = c
	}
	require.Contains(t, got, [2]int{-1, -1})
	assert.Equal(t, [2]int{-8 - 1, -8 - 1}, got[[2]int{-1, -1}].cell)
}

func TestATiledPictureDrawsAQuadATileFromItsPage(t *testing.T) {
	w, win, _, pic, _ := pictureIn(t, geom.V2(50, 50), TexelSize(10, 10), TileTexels(8), PaintAhead(0))
	w.RunFrames(1)
	l := draw.FrontFor(w, win)
	require.Len(t, l.Vertices, 16, "four tiles")
	require.Len(t, l.Batches, 1, "one page, one batch")
	page := pic.pages[0].tex
	rects, whole := page.Changes()
	assert.False(t, whole, "a page is uploaded by the rectangle")
	assert.Len(t, rects, 4)
}

func TestPaintAheadPaintsTilesRoundTheViewAFewAFrame(t *testing.T) {
	w, _, _, pic, _ := pictureIn(t, geom.V2(50, 50), TexelSize(10, 10), TileTexels(8), PaintAhead(2))
	assert.Equal(t, 4+2, pic.Tiles(), "the view's 4, and 2 ahead")
	w.RunFrames(1)
	assert.Equal(t, 4+4, pic.Tiles(), "2 more the next frame")
}

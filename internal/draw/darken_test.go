package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ink is the summed coverage of r in the atlas, 0 to 1 a pixel.
func ink(a *image.Alpha, r image.Rectangle) float64 {
	var sum float64
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			sum += float64(a.AlphaAt(x, y).A) / 255
		}
	}
	return sum
}

func TestStemDarkeningGrowsWithTheRastersSize(t *testing.T) {
	assert.InDelta(t, 0.005*26, darkening(26), 1e-6, "13 points at 2×")
	assert.Greater(t, darkening(48), darkening(26))
}

func TestStemDarkeningAddsToEveryEdgePixelOnly(t *testing.T) {
	a := image.NewAlpha(image.Rect(0, 0, 5, 3))
	a.Pix = []uint8{
		0, 0, 0, 0, 0,
		0, 100, 255, 0, 0,
		0, 0, 0, 0, 0,
	}
	embolden(a, a.Rect, 0.2)
	assert.Equal(t, uint8(100+51), a.AlphaAt(1, 1).A, "a partial pixel gains the offset")
	assert.Equal(t, uint8(255), a.AlphaAt(2, 1).A, "a full one stays full")
	assert.Equal(t, uint8(51), a.AlphaAt(3, 1).A, "an empty one beside a covered one gains it too: the edge moved out")
	assert.Equal(t, uint8(51), a.AlphaAt(2, 0).A)
	assert.Equal(t, uint8(0), a.AlphaAt(4, 1).A, "further out nothing")
	assert.Equal(t, uint8(0), a.AlphaAt(0, 0).A)
}

func TestAGlyphRasterHasRoomToDarkenOutward(t *testing.T) {
	f := Regular.Size(24)
	g := f.glyphAt('l', 0)
	require.False(t, g.rect.Empty())
	for x := g.rect.Min.X; x < g.rect.Max.X; x++ {
		assert.Less(t, f.Atlas.AlphaAt(x, g.rect.Min.Y-1).A, uint8(1), "nothing above the raster")
	}
	assert.Greater(t, ink(f.Atlas, g.rect), 0.0)
}

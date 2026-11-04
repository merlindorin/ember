package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// levels counts r's glyph pixels by coverage: full, and between none and full.
func levels(f *Font, r rune) (int, int) {
	var full, between int
	g := f.glyph(r)
	for y := g.rect.Min.Y; y < g.rect.Max.Y; y++ {
		for x := g.rect.Min.X; x < g.rect.Max.X; x++ {
			switch a := f.Atlas.AlphaAt(x, y).A; a {
			case 0:
			case 255:
				full++
			default:
				between++
			}
		}
	}
	return full, between
}

func TestAPixelFontsGlyphsAreCoveredOrNotAndApartFromTheSmoothedOnes(t *testing.T) {
	smooth, pixel := Regular.Size(13), Regular.PixelSize(13)
	assert.NotSame(t, smooth, pixel, "its own font and atlas")
	assert.Same(t, pixel, Regular.PixelSize(13), "kept like a smoothed size")
	for _, r := range "Ag8@" {
		full, between := levels(pixel, r)
		assert.Zero(t, between, "%q unsmoothed: no partial pixel", r)
		assert.Positive(t, full)
		_, soft := levels(smooth, r)
		assert.Positive(t, soft, "%q smoothed as before", r)
	}
	assert.Equal(t, smooth.measure("Pixel 128", 0), pixel.measure("Pixel 128", 0), "the same advances and metrics")
}

func TestAStyleAsksForAPixelFontWhenPixelated(t *testing.T) {
	assert.Same(t, Regular.PixelSize(10), Style{Size: 10, Pixelated: true}.Font())
	assert.Same(t, Regular.Size(10), Style{Size: 10}.Font())
}

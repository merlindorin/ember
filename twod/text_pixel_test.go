package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/render"
	"github.com/stretchr/testify/assert"
)

func TestPixelatedDrawsATextsGlyphsUnsmoothedAndStillFiltersTextures(t *testing.T) {
	pixel, smooth := NewText("HP", Pixelated(), FontSize(8)), NewText("HP", FontSize(8))
	assert.True(t, pixel.Pixelated())
	assert.False(t, smooth.Pixelated())
	assert.NotSame(t, smooth.style().Font(), pixel.style().Font(), "a pixel text draws from its own unsmoothed font")
	assert.Same(t, Regular().draw().PixelSize(8), pixel.style().Font())
	assert.Equal(t, render.FilterNearest, NewTexture(nil, Pixelated()).Filter, "a texture's nearest filter as before")
}

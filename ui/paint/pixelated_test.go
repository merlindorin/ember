package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
)

func TestAPixelatedCopyKeepsItsSourcesView(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	assert.True(t, pixelated(twod.NewTexture(img, twod.Encoded())).Encoded(), "a UI picture stays encoded")
	assert.False(t, pixelated(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 4, 4)))).Encoded(),
		"a game's sprite stays linear")
}

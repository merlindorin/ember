package pages //nolint:testpackage // white-box: cut is the page tool's own

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASnapshotIsItsBoxCutFromTheCaptureAtItsScale(t *testing.T) {
	capture := image.NewRGBA(image.Rect(0, 0, 20, 20))
	red := color.RGBA{R: 255, A: 255}
	for y := 4; y < 10; y++ {
		for x := 6; x < 14; x++ {
			capture.Set(x, y, red)
		}
	}
	data := cut(capture, geom.Rect{X: 3, Y: 2, W: 4, H: 3}, 2)
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 8, 6), img.Bounds(), "4 by 3 points at 2 pixels a point")
	assert.Equal(t, red, color.RGBAModel.Convert(img.At(0, 0)))
	assert.Equal(t, red, color.RGBAModel.Convert(img.At(7, 5)))
}

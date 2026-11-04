package ui_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
)

func pngOf(w, h int) []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, w, h)))
	return b.Bytes()
}

func TestImagesGivesUrlsAndImgSourcesAFileSystemOfTheirOwn(t *testing.T) {
	art := fstest.MapFS{"icons/bow.png": {Data: pngOf(16, 12)}, "wall.png": {Data: pngOf(8, 8)}}
	tile := ui.Div(ui.ID("tile"))
	icon := ui.Img(ui.Attr("src", "icons/bow.png"))
	d := ui.NewDocument(ui.CSS(`#tile { width: 16px; height: 16px; background-image: url(wall.png); }`), ui.Images(art))
	d.Body().Append(tile, icon)
	d.Layout(geom.V2(200, 100))
	assert.Empty(t, d.Warnings(), "both found in the images' file system")
	assert.Equal(t, float32(16), icon.Box().W, "the img takes its picture's size")
	assert.Equal(t, float32(12), icon.Box().H)
}

func TestAnImageMissingFromEveryFileSystemIsAWarning(t *testing.T) {
	d := ui.NewDocument(ui.Images(fstest.MapFS{}))
	d.Body().Append(ui.Img(ui.Attr("src", "nowhere.png")))
	d.Layout(geom.V2(200, 100))
	assert.NotEmpty(t, d.Warnings())
}

package ui

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestABrokenImageTakesItsPicturesSizeOnceItsSrcNamesOne(t *testing.T) {
	var pic bytes.Buffer
	require.NoError(t, png.Encode(&pic, image.NewRGBA(image.Rect(0, 0, 40, 30))))
	files := fstest.MapFS{"s.css": {Data: []byte("")}, "door.png": {Data: pic.Bytes()}}
	door := Img(Attr("src", "missing.png"), Attr("alt", "A door"))
	d := NewDocument(Stylesheet(files, "s.css"))
	d.Body().Append(door)
	win := platform.NewWindow(platform.Size(300, 200))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	broken := door.Box()
	assert.Greater(t, broken.W, float32(16), "broken, as wide as its icon and its alt text")
	door.SetAttr("src", "door.png")
	w.RunFrames(2)
	assert.Equal(t, geom.V2(40, 30), geom.V2(door.Box().W, door.Box().H), "its picture's size once it names one")
}

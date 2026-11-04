package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"image"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gomono"
)

func TestFacesLoadFromFilesOrMemory(t *testing.T) {
	fromFile, err := LoadFace(fstest.MapFS{"mono.ttf": {Data: gomono.TTF}}, "mono.ttf")
	require.NoError(t, err)
	fromMemory, err := NewFace(gomono.TTF)
	require.NoError(t, err)
	assert.Equal(t, fromFile.Measure("ab", 16), fromMemory.Measure("ab", 16))

	_, err = LoadFace(fstest.MapFS{}, "missing.ttf")
	assert.Error(t, err)
	_, err = LoadFace(fstest.MapFS{"bad.ttf": {Data: []byte("nope")}}, "bad.ttf")
	assert.ErrorContains(t, err, "bad.ttf is not a TrueType font file")
}

func TestOnlyTrueTypeFontsAreSupported(t *testing.T) {
	_, err := LoadFace(fstest.MapFS{"mono.otf": {Data: gomono.TTF}}, "mono.otf")
	assert.ErrorContains(t, err, "mono.otf: fonts are TrueType files, named .ttf")
	_, err = LoadFace(fstest.MapFS{"MONO.TTF": {Data: gomono.TTF}}, "MONO.TTF")
	assert.NoError(t, err, "the extension's case does not matter")
	for data, want := range map[string]string{
		"OTTO\x00\x0a": "OpenType font with CFF outlines",
		"ttcf\x00\x02": "font collection",
		"wOFF\x00\x01": "web font",
		"wOF2\x00\x01": "web font",
		"ab":           "is not a TrueType font file",
	} {
		_, faceErr := NewFace([]byte(data))
		assert.ErrorContains(t, faceErr, want)
		assert.ErrorContains(t, faceErr, "TrueType")
	}
}

func TestABitmapFontNeedsItsCellsAndDefaultsToASCII(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 95*4, 4))
	_, err := NewBitmapFont(img)
	assert.Error(t, err, "without Cell")
	font, err := NewBitmapFont(img, Cell(4, 4))
	require.NoError(t, err)
	assert.Len(t, font.glyphs, 95, "the printable ASCII characters")
	assert.Contains(t, font.glyphs, '~')
}

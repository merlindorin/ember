package ui

import (
	"bytes"
	"image"
	_ "image/png" // ui.Images and url() backgrounds decode PNG
	"io/fs"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/css"
	"github.com/merlindorin/ember/ui/internal/box"
)

// imageBytes is the file url names: one of css.DefaultImages, else a path
// in the stylesheets' file systems, the last added first.
func (d *Document) imageBytes(url string) ([]byte, bool) {
	if b, ok := css.DefaultImages()[url]; ok {
		return b, true
	}
	for i := len(d.files) - 1; i >= 0; i-- {
		if b, err := fs.ReadFile(d.files[i], url); err == nil {
			return b, true
		}
	}
	return nil, false
}

// image is the texture url names, decoded once; nil when there is none,
// with a warning.
func (d *Document) image(url string) *twod.Texture {
	if t, ok := d.images[url]; ok {
		return t
	}
	var tex *twod.Texture
	if b, ok := d.imageBytes(url); ok {
		if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
			tex = twod.NewTexture(img, twod.Encoded())
		} else {
			d.warn(err)
		}
	} else {
		d.warn(&missingImageError{url})
	}
	if d.images == nil {
		d.images = map[string]*twod.Texture{}
	}
	d.images[url] = tex
	return tex
}

type missingImageError struct{ url string }

func (e *missingImageError) Error() string { return "ui: no image " + e.url }

// ImageSize is the natural size of the image url names, one point per
// pixel, for layout's replaced elements (box.Images).
func (d *Document) ImageSize(url string) (geom.Vec2, bool) {
	t := d.image(url)
	if t == nil {
		return geom.Vec2{}, false
	}
	return t.Size(), true
}

var _ box.Images = (*Document)(nil)

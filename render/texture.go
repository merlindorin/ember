package render

import "image"

// Texture is an image a material samples, in sRGB. It is plain data: the
// presenter uploads it the first time it is drawn and again after Touch.
type Texture struct {
	Image  image.Image
	Filter Filter
	// Mipmaps smooths minification; ignored with FilterNearest.
	Mipmaps bool
	Repeat  bool
	// Encoded filters the image in encoded sRGB, as CSS filters images,
	// rather than in linear light as sprites are (decision 0056): ui's
	// pictures.
	Encoded bool

	version uint64
	// rects are the rectangles touched since the last upload; whole is
	// set by Touch, the whole image.
	rects []image.Rectangle
	whole bool
}

// Version changes whenever Touch or TouchRect is called; GPU caches
// compare it.
func (t *Texture) Version() uint64 { return t.version }

// Touch marks the image as modified so the presenter re-uploads it.
func (t *Texture) Touch() { t.version, t.whole = t.version+1, true }

// TouchRect marks r of the image as modified: the presenter re-uploads
// only the rectangles touched since it last uploaded, unless Touch was
// called too.
func (t *Texture) TouchRect(r image.Rectangle) {
	t.version++
	if !r.Empty() {
		t.rects = append(t.rects, r)
	}
}

// Changes are what changed since Uploaded was last called: the
// rectangles TouchRect gave, or whole when Touch was called.
func (t *Texture) Changes() ([]image.Rectangle, bool) { return t.rects, t.whole }

// Uploaded tells the texture its presenter has its image as it is now.
func (t *Texture) Uploaded() { t.rects, t.whole = t.rects[:0], false }

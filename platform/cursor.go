package platform

import (
	"image"
	"sync"
)

// customCursors is where NewCursor's cursors start, past the standard
// shapes.
const customCursors Cursor = 1 << 16

// cursorImage is a cursor made by NewCursor.
type cursorImage struct {
	img, hiDPI image.Image
	hotX, hotY int
}

// CursorOption tunes a cursor made by NewCursor.
type CursorOption func(*cursorImage)

// HiDPI gives the cursor a picture drawn at twice its size, which screens
// with two pixels a point show instead: on the web and on Windows and
// Linux. macOS shows the plain picture, scaled.
func HiDPI(img image.Image) CursorOption { return func(c *cursorImage) { c.hiDPI = img } }

var cursorImages struct { //nolint:gochecknoglobals // NewCursor's cursors are values every window shares
	sync.Mutex
	list []*cursorImage
}

// NewCursor makes a cursor of img, its hot spot (the point that clicks)
// hotX, hotY pixels from img's top-left; img's pixels are points. Pass it
// to Window.SetCursor like the standard shapes. It is a plain value, safe
// to make before any window opens: each window makes its own system
// cursor the first time it shows it. Make each cursor once, as a game
// keeps a texture.
func NewCursor(img image.Image, hotX, hotY int, opts ...CursorOption) Cursor {
	c := &cursorImage{img: img, hotX: hotX, hotY: hotY}
	for _, o := range opts {
		o(c)
	}
	cursorImages.Lock()
	defer cursorImages.Unlock()
	cursorImages.list = append(cursorImages.list, c)
	return customCursors + Cursor(len(cursorImages.list)-1)
}

func (c Cursor) made() *cursorImage {
	if c < customCursors {
		return nil
	}
	cursorImages.Lock()
	defer cursorImages.Unlock()
	if i := int(c - customCursors); i < len(cursorImages.list) {
		return cursorImages.list[i]
	}
	return nil
}

// Image is the picture of a cursor made by NewCursor, nil for a standard
// shape.
func (c Cursor) Image() image.Image {
	if m := c.made(); m != nil {
		return m.img
	}
	return nil
}

// Hot is the hot spot of a cursor made by NewCursor, x then y, in its
// picture's pixels from the top-left.
func (c Cursor) Hot() (int, int) {
	if m := c.made(); m != nil {
		return m.hotX, m.hotY
	}
	return 0, 0
}

// HiDPI is the cursor's picture at twice its size, nil without one.
func (c Cursor) HiDPI() image.Image {
	if m := c.made(); m != nil {
		return m.hiDPI
	}
	return nil
}

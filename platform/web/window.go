//go:build js

package web

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"syscall/js"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
)

// canvas is the primary window's backend: the page's canvas. A page has
// no window frame to move, resize or decorate, so those do nothing.
type canvas struct {
	el js.Value
	in *input.State
	// pasted is the text of the page's last paste, which Clipboard gives:
	// a page reads the clipboard only when the player pastes.
	pasted string
}

//nolint:gochecknoglobals // a lookup table
var cursors = map[platform.Cursor]string{
	platform.CursorArrow:      "default",
	platform.CursorHidden:     "none",
	platform.CursorText:       "text",
	platform.CursorHand:       "pointer",
	platform.CursorResizeEW:   "ew-resize",
	platform.CursorResizeNS:   "ns-resize",
	platform.CursorResizeNWSE: "nwse-resize",
}

func newCanvas(ls *listeners, el js.Value, in *input.State) *canvas {
	c := &canvas{el: el, in: in}
	ls.on(js.Global(), "paste", func(e js.Value) {
		if data := e.Get("clipboardData"); !data.IsNull() && !data.IsUndefined() {
			c.pasted = data.Call("getData", "text/plain").String()
		}
	})
	return c
}

func (c *canvas) SetTitle(title string) { js.Global().Get("document").Set("title", title) }

func (c *canvas) SetCursor(cur platform.Cursor) {
	css, ok := cursors[cur]
	if !ok {
		css = pictureCursor(cur)
		cursors[cur] = css
	}
	c.el.Get("style").Set("cursor", css)
}

// pictureCursor is CSS for a cursor from NewCursor: its picture as a data
// URL, beside its HiDPI one in an image-set, the arrow if the browser
// can't show it.
func pictureCursor(cur platform.Cursor) string {
	x, y := cur.Hot()
	src := dataURL(cur.Image())
	if big := cur.HiDPI(); big != nil {
		src = fmt.Sprintf("image-set(%s 1x, %s 2x)", src, dataURL(big))
	}
	return fmt.Sprintf("%s %d %d, auto", src, x, y)
}

func dataURL(img image.Image) string {
	var b bytes.Buffer
	_ = png.Encode(&b, img) // into memory, so it fails only on a broken image
	return `url("data:image/png;base64,` + base64.StdEncoding.EncodeToString(b.Bytes()) + `")`
}

func (c *canvas) Clipboard() string { return c.pasted }

func (c *canvas) SetClipboard(text string) {
	if cb := js.Global().Get("navigator").Get("clipboard"); !cb.IsUndefined() {
		cb.Call("writeText", text)
	}
	c.pasted = text
}

func (c *canvas) SetPosition(_, _ int)         {}
func (c *canvas) SetSize(_, _ int)             {}
func (c *canvas) SetResizable(_ bool)          {}
func (c *canvas) SetAlwaysOnTop(_ bool)        {}
func (c *canvas) SetSizeLimits(_, _ geom.Vec2) {}
func (c *canvas) SetAspectRatio(_, _ int)      {}
func (c *canvas) Maximize()                    {}
func (c *canvas) Minimize()                    {}
func (c *canvas) Restore()                     {}
func (c *canvas) SetIcon(_ []image.Image)      {}
func (c *canvas) WindowPosition() (int, int)   { return 0, 0 }
func (c *canvas) ScreenCursor() (float64, float64) {
	m := c.in.Mouse()
	return float64(m.X), float64(m.Y)
}

// SetFullscreen asks for the whole page full screen, not the canvas, which
// the browser would stretch past the size the game reads; the runner then
// fits the canvas to the page. Browsers grant it only during a click or a
// key.
func (c *canvas) SetFullscreen(on bool) {
	doc := js.Global().Get("document")
	if on {
		doc.Get("documentElement").Call("requestFullscreen")
	} else if !doc.Get("fullscreenElement").IsNull() {
		doc.Call("exitFullscreen")
	}
}

var _ platform.Backend = (*canvas)(nil)

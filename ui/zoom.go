package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

// Zoom draws the Document f times its size, as a browser's zoom does: it
// lays out in a viewport 1/f of its window, text is rasterised at the
// zoomed size, and the pointer and automation's bounds are scaled back,
// so clicks land where things are drawn. 1, the default, is no zoom.
func Zoom(f float32) DocumentOption { return func(d *Document) { d.SetZoom(f) } }

// Zoom is the Document's zoom, 1 unless set.
func (d *Document) Zoom() float32 {
	if d.zoom <= 0 {
		return 1
	}
	return d.zoom
}

// SetZoom changes the Document's zoom; the next frame lays it out again.
func (d *Document) SetZoom(f float32) {
	if f > 0 && f != d.Zoom() {
		d.zoom = f
		d.replay.repaint = true
	}
}

// toWindow is r, in the Document's points, in its window's.
func (d *Document) toWindow(r geom.Rect) geom.Rect {
	z := d.Zoom()
	return geom.Rect{X: r.X * z, Y: r.Y * z, W: r.W * z, H: r.H * z}
}

// mouse is the pointer in the Document's points.
func (d *Document) mouse(c *twod.Canvas) geom.Vec2 { return c.Mouse().Scale(1 / d.Zoom()) }

// zoomed runs paint with the canvas scaled by the zoom.
func (d *Document) zoomed(c *twod.Canvas, paint func()) {
	z := d.Zoom()
	if z == 1 {
		paint()
		return
	}
	t := c.Transform()
	c.SetTransform(t.Mul(geom.Transform2D{X: geom.V2(z, 0), Y: geom.V2(0, z)}))
	paint()
	c.SetTransform(t)
}

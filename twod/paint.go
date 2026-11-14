package twod

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
)

// The canvas's drawing, in the space of what is being drawn: Draw gets a
// canvas already placed, turned, scaled and tinted like its entity.

// Rect fills a rectangle.
func (u *Canvas) Rect(r geom.Rect, c color.Color) { u.list.Rect(r, c) }

// RoundRect fills an anti-aliased rounded rectangle.
func (u *Canvas) RoundRect(r geom.Rect, radius float32, c color.Color) {
	u.list.RoundRect(r, radius, c)
}

// RoundRectCorners fills a rounded rectangle whose corners have their own
// radii: top-left, top-right, bottom-right, bottom-left.
func (u *Canvas) RoundRectCorners(r geom.Rect, radii [4]float32, c color.Color) {
	u.list.RoundRectCorners(r, radii, c)
}

// BorderCorners strokes a rounded rectangle outline whose corners have
// their own radii, in RoundRectCorners's order.
func (u *Canvas) BorderCorners(r geom.Rect, radii [4]float32, thickness float32, c color.Color) {
	u.list.BorderCorners(r, radii, thickness, c)
}

// Circle fills an anti-aliased disc.
func (u *Canvas) Circle(center geom.Vec2, radius float32, c color.Color) {
	u.list.Circle(center, radius, c)
}

// Ring strokes a circle outline of the given thickness.
func (u *Canvas) Ring(center geom.Vec2, radius, thickness float32, c color.Color) {
	u.list.Ring(center, radius, thickness, c)
}

// Orb draws a glass globe filled with animated liquid up to fill (0..1),
// the classic action-RPG health and mana display.
func (u *Canvas) Orb(center geom.Vec2, radius, fill float32, liquid color.Color) {
	u.list.Orb(center, radius, fill, liquid)
}

// Border strokes a rounded rectangle outline.
func (u *Canvas) Border(r geom.Rect, radius, thickness float32, c color.Color) {
	u.list.Border(r, radius, thickness, c)
}

// Line draws a segment of the given width.
func (u *Canvas) Line(a, b geom.Vec2, width float32, c color.Color) { u.list.Line(a, b, width, c) }

// Glow fades c out around a rounded rectangle, from its edge to spread
// points beyond it, leaving the inside clear: a halo around a focused
// card, or a soft shadow.
func (u *Canvas) Glow(r geom.Rect, radius, spread float32, c color.Color) {
	u.list.Glow(r, radius, spread, c)
}

// Gradient fills a rectangle with a vertical gradient.
func (u *Canvas) Gradient(r geom.Rect, top, bottom color.Color) { u.list.Gradient(r, top, bottom) }

// Triangles fills triangles of points, three indices each.
func (u *Canvas) Triangles(points []geom.Vec2, indices []int, c color.Color) {
	u.list.Triangles(points, indices, c)
}

// TrianglesColored fills triangles of points, three indices each, each
// point in its own colour from colors, blended across the triangles, so a
// shape can fade at its rim: a line-of-sight polygon whose edge goes
// transparent. A triangle with an index past points or colors is skipped.
func (u *Canvas) TrianglesColored(points []geom.Vec2, colors []color.Color, indices []int) {
	u.list.TrianglesColored(points, colors, indices)
}

// Image draws the uv part (0..1) of tex over r, tinted.
func (u *Canvas) Image(r geom.Rect, tex *Texture, uv geom.Rect, tint color.Color) {
	if tex != nil {
		u.list.Image(r, &tex.Texture, uv, tint)
	}
}

// NinePatch stretches tex over r but keeps its margins, in texture
// pixels, unscaled: corners stay sharp, edges stretch along, the centre
// both ways.
func (u *Canvas) NinePatch(r geom.Rect, tex *Texture, left, top, right, bottom float32, tint color.Color) {
	if tex != nil && tex.Image != nil {
		u.list.NinePatch(r, &tex.Texture, left, top, right, bottom, tint)
	}
}

// PushClip bounds drawing to r, within the current clip, until PopClip.
func (u *Canvas) PushClip(r geom.Rect) { u.list.PushClip(r) }

// PushClipRounded bounds drawing to r with its corners rounded by radii
// (top-left, top-right, bottom-right, bottom-left), within the current
// clip, until PopClip. Only the innermost rounded clip rounds; one around
// it then clips as its rect, and under a rotation or a skew it clips to
// r's bounds, square (decision 0058).
func (u *Canvas) PushClipRounded(r geom.Rect, radii [4]float32) { u.list.PushClipRounded(r, radii) }

func (u *Canvas) PopClip() { u.list.PopClip() }

// Transform is the space drawing happens in, from the canvas's origin.
func (u *Canvas) Transform() geom.Transform2D { return u.list.Transform() }

// SetTransform changes the space what is drawn next is given in.
func (u *Canvas) SetTransform(t geom.Transform2D) { u.list.SetTransform(t) }

// Time is the elapsed time, for animated drawing.
func (u *Canvas) Time() float32 { return u.list.Time() }

package twod

import "github.com/merlindorin/ember/geom"

// cullMargin is how far outside a view, in its points, an entity still
// counts as in it: room for what is drawn past its bounds, such as a
// shadow or a glow.
const cullMargin = 16

// bounded is an entity that knows the rectangle it draws in its own space.
// Entities that don't are always drawn.
type bounded interface {
	drawBounds() geom.Rect
}

func (s *Sprite) drawBounds() geom.Rect { return s.Rect() }

func (p *NinePatch) drawBounds() geom.Rect {
	return geom.Rect{X: -p.size.X * p.pivot.X, Y: -p.size.Y * p.pivot.Y, W: p.size.X, H: p.size.Y}
}

func (l *Line) drawBounds() geom.Rect { return pointsBounds(l.points, l.width) }

func (p *Polygon) drawBounds() geom.Rect { return pointsBounds(p.points, 0) }

// pointsBounds is the rectangle around points, grown by pad on each side.
func pointsBounds(points []geom.Vec2, pad float32) geom.Rect {
	if len(points) == 0 {
		return geom.Rect{}
	}
	lo, hi := points[0], points[0]
	for _, p := range points[1:] {
		lo, hi = geom.V2(min(lo.X, p.X), min(lo.Y, p.Y)), geom.V2(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	return geom.Rect{X: lo.X - pad, Y: lo.Y - pad, W: hi.X - lo.X + 2*pad, H: hi.Y - lo.Y + 2*pad}
}

// cull is what a window's cameras show this frame: an entity is drawn when
// one camera that draws its layer has it in view.
type cull struct {
	screen geom.Vec2
	views  []*Camera
	// set is whether reset made it this frame's.
	set bool
	// areas, spans and kept are collect's scratch, their memory reused:
	// pick's areas, and a SortByY parent's children's stretches.
	areas []geom.Rect
	spans []ySpan
	kept  []entry
}

// reset makes c the views of a window of screen points, through its camera
// cam, if any, and its cameras into textures.
func (c *cull) reset(screen geom.Vec2, cam *Camera, feeds []*Camera) {
	c.screen, c.set = screen, true
	clear(c.views)
	c.views = c.views[:0]
	if cam != nil {
		c.views = append(c.views, cam)
	}
	c.views = append(c.views, feeds...)
	clear(c.kept)
}

// visible reports whether en shows in any view: always when it is on a
// fixed layer, the UI's included, or doesn't know its bounds.
func (c *cull) visible(en entry) bool {
	if en.fixed {
		return true
	}
	b, ok := en.e.(bounded)
	if !ok {
		return true
	}
	local := b.drawBounds()
	if len(c.views) == 0 || c.views[0].into != nil {
		if inView(en.global, local, geom.Identity2D(), c.screen) {
			return true
		}
	}
	for _, cam := range c.views {
		if !cam.shows(en) {
			continue
		}
		if inView(en.global, local, cam.transform(cam.viewport, en.parallax), cam.viewport) {
			return true
		}
	}
	return false
}

// mapViews is what each camera in cams, nil for none, shows of e's own
// space, or the window's screen when there is no camera or e is fixed.
func mapViews(out []geom.Rect, e Entity, cams []*Camera, screen geom.Vec2, alpha float32) []geom.Rect {
	global := drawnGlobal(e, alpha)
	parallax, fixed := float32(1), false
	for p := e.GetSceneNode().ParentNode(); p != nil; p = p.ParentNode() {
		if l, ok := p.Entity().(*Layer); ok {
			parallax, fixed = l.parallax, l.fixed
			break
		}
	}
	for _, cam := range cams {
		if cam == nil || fixed {
			continue
		}
		out = append(out, seen(cam.transform(cam.viewport, parallax).Mul(global), cam.viewport))
	}
	if len(out) == 0 || cams[0] == nil {
		out = append(out, seen(global, screen))
	}
	return out
}

// seen is the rectangle of a space that a viewport of size shows through
// view, which maps that space to the viewport.
func seen(view geom.Transform2D, size geom.Vec2) geom.Rect {
	inv := view.Inverse()
	lo, hi := geom.V2(1e30, 1e30), geom.V2(-1e30, -1e30)
	for _, p := range []geom.Vec2{{}, {X: size.X}, {Y: size.Y}, size} {
		q := inv.Xform(p)
		lo, hi = geom.V2(min(lo.X, q.X), min(lo.Y, q.Y)), geom.V2(max(hi.X, q.X), max(hi.Y, q.Y))
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

// inView reports whether local, placed by global and seen through view,
// overlaps a viewport of size points, grown by cullMargin.
func inView(global geom.Transform2D, local geom.Rect, view geom.Transform2D, size geom.Vec2) bool {
	t := view.Mul(global)
	lo, hi := geom.V2(1e30, 1e30), geom.V2(-1e30, -1e30)
	for _, p := range []geom.Vec2{
		{X: local.X, Y: local.Y},
		{X: local.X + local.W, Y: local.Y},
		{X: local.X, Y: local.Y + local.H},
		{X: local.X + local.W, Y: local.Y + local.H},
	} {
		q := t.Xform(p)
		lo, hi = geom.V2(min(lo.X, q.X), min(lo.Y, q.Y)), geom.V2(max(hi.X, q.X), max(hi.Y, q.Y))
	}
	return hi.X >= -cullMargin && hi.Y >= -cullMargin && lo.X <= size.X+cullMargin && lo.Y <= size.Y+cullMargin
}

// viewed is the part of what c draws in, in its own space, that its window
// shows: what a map draws of itself.
func viewed(c *Canvas) geom.Rect {
	screen := c.Screen()
	inv := c.Transform().Inverse()
	lo, hi := geom.V2(1e30, 1e30), geom.V2(-1e30, -1e30)
	for _, p := range []geom.Vec2{geom.V2(0, 0), geom.V2(screen.W, 0), geom.V2(0, screen.H), geom.V2(screen.W, screen.H)} {
		q := inv.Xform(p)
		lo, hi = geom.V2(min(lo.X, q.X), min(lo.Y, q.Y)), geom.V2(max(hi.X, q.X), max(hi.Y, q.Y))
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

// pick is the children of e that may show, when e is a layer that files
// them in a grid: those near what its views show, in their order.
func (c *cull) pick(e Entity, local geom.Transform2D) ([]*filed, bool) {
	l, ok := e.(*Layer)
	if !ok {
		return nil, false
	}
	g := l.gridded()
	if g == nil {
		return nil, false
	}
	areas := c.areas[:0]
	if len(c.views) == 0 || c.views[0].into != nil {
		areas = append(areas, seenGrown(local, c.screen, cullMargin))
	}
	for _, cam := range c.views {
		if !cam.shows(entry{e: l}) {
			continue
		}
		areas = append(areas, seenGrown(cam.transform(cam.viewport, l.parallax).Mul(local), cam.viewport, cullMargin))
	}
	c.areas = areas
	return g.visit(areas), true
}

// seenGrown is what seen is of a viewport grown by margin on every side.
func seenGrown(view geom.Transform2D, size geom.Vec2, margin float32) geom.Rect {
	inv := view.Inverse()
	lo, hi := geom.V2(1e30, 1e30), geom.V2(-1e30, -1e30)
	for _, p := range [4]geom.Vec2{
		{X: -margin, Y: -margin},
		{X: size.X + margin, Y: -margin},
		{X: -margin, Y: size.Y + margin},
		{X: size.X + margin, Y: size.Y + margin},
	} {
		q := inv.Xform(p)
		lo, hi = geom.V2(min(lo.X, q.X), min(lo.Y, q.Y)), geom.V2(max(hi.X, q.X), max(hi.Y, q.Y))
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

package main

import (
	"fmt"
	"math"
	"runtime"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
)

// The colours shapes are drawn in: collision orange, light-only blue.
//
//nolint:gochecknoglobals // fixed values every draw reads; Go has no constant colours
var (
	collisionColor = color.Hex("#ff9a3c")
	shadowColor    = color.Hex("#5ab4ff")
	bodyColor      = color.Hex("#b07cff")
)

func shapeColor(s Shape) color.Color {
	switch {
	case s.Shadow:
		return shadowColor
	case s.Body:
		return bodyColor
	}
	return collisionColor
}

// picture draws tile id of tool's tileset into r.
func (t *Tool) picture(c *twod.Canvas, id int, r geom.Rect) {
	if t.tiles == nil {
		return
	}
	tex, src := t.tiles.Picture(id + 1)
	if tex == nil {
		return
	}
	size := tex.Size()
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	c.Image(r, tex, geom.Rect{X: src.X / size.X, Y: src.Y / size.Y, W: src.W / size.X, H: src.H / size.Y}, color.White)
}

// slot is a widget drawn in a box the shell lays out, placed there each
// frame: the tile grid, the editor and the preview pane.
type slot struct {
	twod.Item
	rect geom.Rect
}

// Rect is the box the widget stands in, in window points.
func (s *slot) Rect() geom.Rect { return s.rect }

func (s *slot) place(r geom.Rect) {
	s.rect = r
	s.SetPosition(geom.V2(r.X, r.Y))
}

func (s *slot) BeginDraw(c *twod.Canvas) func() {
	c.SetSize(geom.V2(s.rect.W, s.rect.H))
	return func() {}
}

// tileGrid shows the tileset's tiles as the game draws them, two points
// an art pixel, in rows; a click picks one.
type tileGrid struct {
	slot
	tool   *Tool
	scroll float32
}

func (g *tileGrid) Draw(c *twod.Canvas) {
	size := c.Size()
	c.Rect(geom.Rect{W: size.X, H: size.Y}, color.Hex("#1b1d22"))
	in := c.Input()
	mouse := c.Transform().Inverse().Xform(in.Mouse())
	if mouse.X >= 0 && mouse.Y >= 0 && mouse.X < size.X && mouse.Y < size.Y {
		g.scroll = max(g.scroll-in.Wheel().Y*24, 0)
	}
	c.PushClip(geom.Rect{W: size.X, H: size.Y})
	x, y, row := float32(8), float32(8)-g.scroll, float32(0)
	for _, id := range g.tool.ed.ids {
		s := tileSize(g.tool.ed.doc, id).Scale(2)
		if x+s.X > size.X-8 && x > 8 {
			x, y, row = 8, y+row+8, 0
		}
		r := geom.Rect{X: x, Y: y, W: s.X, H: s.Y}
		name := fmt.Sprintf("tile-%d", id)
		if c.Interact(r, name).Clicked {
			if in.Down(input.KeyLeftShift) || in.Down(input.KeyRightShift) || command(in) {
				g.tool.mark(id)
			} else {
				g.tool.pick(id)
			}
		}
		c.Rect(r, color.Hex("#2a2d34"))
		g.tool.picture(c, id, r)
		switch {
		case id == g.tool.ed.tile.ID:
			c.Border(r, 0, 2, color.Hex("#ffd34a"))
		case g.tool.ed.marked[id]:
			c.Border(r, 0, 2, color.Hex("#5ab4ff"))
		}
		c.Record(name, "button", fmt.Sprintf("tile %d", id), nil, true, r, "press")
		x, row = x+s.X+8, max(row, s.Y)
	}
	c.PopClip()
}

// drag is a shape being moved or resized in the editor: from where, and
// which corner or point, -1 for a move.
type drag struct {
	on     bool
	start  geom.Vec2
	handle int
	shape  Shape
	fit    func(float32) float32
}

// tileEditor shows the picked tile magnified, its shapes over it, and
// lets the mouse select, move and resize them, snapped to art pixels.
type tileEditor struct {
	slot
	tool *Tool
	drag drag
	// lastPress and lastAt are the last press's time and place, for a
	// double click.
	lastPress float32
	lastAt    geom.Vec2
	// zoom multiplies the zoom that fits the tile, the wheel's; pan moves
	// it, a middle drag's.
	zoom float32
	pan  geom.Vec2
}

// frame is where the tile's corner stands in the editor and its zoom,
// ext, the tile and its shapes, fitting the editor's size.
func (e *tileEditor) frame(size geom.Vec2, ext geom.Rect) (geom.Vec2, float32) {
	zoom := float32(math.Floor(float64(min((size.X-32)/max(ext.W, 1), (size.Y-32)/max(ext.H, 1)))))
	zoom = max(min(zoom, 24), 1)
	if e.zoom > 0 {
		zoom *= e.zoom
	}
	return geom.V2((size.X-ext.W*zoom)/2-ext.X*zoom, (size.Y-ext.H*zoom)/2-ext.Y*zoom).Add(e.pan), zoom
}

// view zooms by the wheel and pans by a middle drag over the area of size.
func (e *tileEditor) view(c *twod.Canvas, size geom.Vec2) {
	in := c.Input()
	mouse := c.Transform().Inverse().Xform(in.Mouse())
	if mouse.X < 0 || mouse.Y < 0 || mouse.X >= size.X || mouse.Y >= size.Y {
		return
	}
	if w := in.Wheel().Y; w != 0 {
		if e.zoom == 0 {
			e.zoom = 1
		}
		e.zoom = min(max(e.zoom*float32(math.Pow(1.25, float64(w))), 0.25), 8)
	}
	if in.Down(input.MouseMiddle) {
		e.pan = e.pan.Add(in.MouseDelta())
	}
}

// handles are a shape's handles, in the tile's pixels: its box's four
// corners, or a polygon's points.
func handles(s Shape) []geom.Vec2 {
	if s.Kind == Polygon {
		out := make([]geom.Vec2, len(s.Points))
		for i, p := range s.Points {
			out[i] = geom.V2(s.X+p.X, s.Y+p.Y)
		}
		return out
	}
	b := s.Bounds()
	return []geom.Vec2{{X: b.X, Y: b.Y}, {X: b.X + b.W, Y: b.Y}, {X: b.X + b.W, Y: b.Y + b.H}, {X: b.X, Y: b.Y + b.H}}
}

// dragged is s as the drag leaves it, the mouse at p, in the tile's
// pixels, snapped.
func (d drag) dragged(p geom.Vec2) Shape {
	s := d.shape.clone()
	delta := geom.V2(d.fit(p.X-d.start.X), d.fit(p.Y-d.start.Y))
	switch {
	case d.handle < 0:
		s.X, s.Y = s.X+delta.X, s.Y+delta.Y
	case s.Kind == Polygon:
		s.Points[d.handle] = s.Points[d.handle].Add(delta)
	default:
		b := s.Bounds()
		x0, y0, x1, y1 := b.X, b.Y, b.X+b.W, b.Y+b.H
		switch d.handle {
		case 0:
			x0, y0 = x0+delta.X, y0+delta.Y
		case 1:
			x1, y0 = x1+delta.X, y0+delta.Y
		case 2:
			x1, y1 = x1+delta.X, y1+delta.Y
		case 3:
			x0, y1 = x0+delta.X, y1+delta.Y
		}
		s.X, s.Y, s.W, s.H = min(x0, x1), min(y0, y1), max(abs(x1-x0), 1), max(abs(y1-y0), 1)
	}
	return s
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }

func (e *tileEditor) Draw(c *twod.Canvas) {
	ed := e.tool.ed
	size := c.Size()
	c.PushClip(geom.Rect{W: size.X, H: size.Y})
	defer c.PopClip()
	c.Rect(geom.Rect{W: size.X, H: size.Y}, color.Hex("#14161a"))
	if ed.path == "" {
		return
	}
	e.view(c, size)
	origin, zoom := e.frame(size, ed.extent())
	on := placement{origin, zoom}
	e.drawTile(c, on)

	in := c.Input()
	mouse := c.Transform().Inverse().Xform(in.Mouse())
	p := mouse.Sub(origin).Scale(1 / zoom)
	inside := mouse.X >= 0 && mouse.Y >= 0 && mouse.X < size.X && mouse.Y < size.Y
	if inside && in.Pressed(input.MouseLeft) {
		e.press(p, zoom, in.Down(input.KeyLeftAlt) || in.Down(input.KeyRightAlt), c.World().Time().RealElapsed)
	}
	shapes := ed.tile.Shapes
	if e.drag.on && ed.selected >= 0 && ed.selected < len(shapes) {
		shapes = append([]Shape(nil), shapes...)
		shapes[ed.selected] = e.drag.dragged(p)
		room := ed.room(e.drag.shape)
		if e.drag.handle < 0 && shapes[ed.selected].Kind == Polygon {
			shapes[ed.selected] = ed.inside(shapes[ed.selected], room)
		}
		shapes[ed.selected] = ed.clamp(shapes[ed.selected], room)
		if in.Released(input.MouseLeft) {
			e.commit(shapes[ed.selected])
		}
	}
	e.drawShapes(c, shapes, on)
	c.Record(
		"tile-editor",
		"canvas",
		fmt.Sprintf("tile %d, %d shapes", ed.tile.ID, len(ed.tile.Shapes)),
		len(ed.tile.Shapes),
		true,
		geom.Rect{W: size.X, H: size.Y},
	)
}

// placement puts tile pixels on the editor's canvas: origin is where the
// tile's top-left lands, zoom the points per tile pixel.
type placement struct {
	origin geom.Vec2
	zoom   float32
}

func (p placement) at(v geom.Vec2) geom.Vec2 { return p.origin.Add(v.Scale(p.zoom)) }

func (p placement) box(r geom.Rect) geom.Rect {
	return geom.Rect{X: p.origin.X + r.X*p.zoom, Y: p.origin.Y + r.Y*p.zoom, W: r.W * p.zoom, H: r.H * p.zoom}
}

// drawTile draws the tile on a checkerboard, with the pixel grid when it is
// on and the tile is zoomed enough to see it.
func (e *tileEditor) drawTile(c *twod.Canvas, on placement) {
	ed := e.tool.ed
	origin, zoom := on.origin, on.zoom
	tile := geom.Rect{W: ed.size.X, H: ed.size.Y}
	for y := float32(0); y < ed.size.Y; y++ {
		for x := float32(0); x < ed.size.X; x++ {
			shade := color.Hex("#2a2d34")
			if int(x+y)%2 == 0 {
				shade = color.Hex("#23262c")
			}
			c.Rect(on.box(geom.Rect{X: x, Y: y, W: 1, H: 1}), shade)
		}
	}
	e.tool.picture(c, ed.tile.ID, on.box(tile))
	if e.tool.checked("grid") && zoom >= 4 {
		line := color.RGBA(1, 1, 1, 0.08)
		for x := float32(0); x <= ed.size.X; x++ {
			c.Rect(geom.Rect{X: origin.X + x*zoom, Y: origin.Y, W: 1, H: ed.size.Y * zoom}, line)
		}
		for y := float32(0); y <= ed.size.Y; y++ {
			c.Rect(geom.Rect{X: origin.X, Y: origin.Y + y*zoom, W: ed.size.X * zoom, H: 1}, line)
		}
	}
}

// drawShapes draws the tile's shapes, the selected one with its handles.
func (e *tileEditor) drawShapes(c *twod.Canvas, shapes []Shape, on placement) {
	ed := e.tool.ed
	effective := e.tool.checked("effective")
	for i, s := range shapes {
		col := shapeColor(s)
		b := on.box(s.Bounds())
		switch s.Kind {
		case Rect:
			c.Rect(b, col.Alpha(0.25))
			c.Border(b, 0, 2, col)
		case Ellipse:
			if effective {
				c.Border(b, 0, 1, col.Alpha(0.45))
			}
			outline(c, ellipsePoints(b), col)
		case Polygon:
			if effective {
				c.Border(b, 0, 1, col.Alpha(0.45))
			}
			var pts []geom.Vec2
			for _, h := range handles(s) {
				pts = append(pts, on.at(h))
			}
			outline(c, pts, col)
		case Point:
			c.Circle(on.at(geom.V2(s.X, s.Y)), 4, col)
		}
		if i == ed.selected {
			hs := handles(s)
			for j, h := range hs {
				q := on.at(h)
				handle := color.White
				if s.Kind == Polygon && j == ed.point {
					handle = color.Hex("#ffd34a")
				}
				c.Rect(geom.Rect{X: q.X - 4, Y: q.Y - 4, W: 8, H: 8}, handle)
			}
			if s.Kind == Polygon {
				for j := range hs {
					m := on.at(hs[j].Add(hs[(j+1)%len(hs)]).Scale(0.5))
					c.Circle(m, 5, color.RGBA(1, 1, 1, 0.35))
					c.Rect(geom.Rect{X: m.X - 3, Y: m.Y - 0.5, W: 6, H: 1}, color.White)
					c.Rect(geom.Rect{X: m.X - 0.5, Y: m.Y - 3, W: 1, H: 6}, color.White)
				}
			}
		}
	}
}

// press starts a drag on a handle of the selected shape, else selects the
// shape under p, the topmost, and starts moving it. On a selected
// polygon, a point's handle selects the point, Alt removing it; an edge's
// middle handle, or a double click on the edge, puts a point there.
func (e *tileEditor) press(p geom.Vec2, zoom float32, alt bool, now float32) {
	ed := e.tool.ed
	twice := now-e.lastPress <= 0.4 && e.lastAt.Sub(p).Len()*zoom <= 6
	e.lastPress, e.lastAt = now, p
	if ed.selected >= 0 && ed.selected < len(ed.tile.Shapes) && e.pressSelected(p, zoom, alt, twice) {
		return
	}
	for i := len(ed.tile.Shapes) - 1; i >= 0; i-- {
		s := ed.tile.Shapes[i]
		b := s.Bounds()
		if p.X >= b.X && p.Y >= b.Y && p.X <= b.X+b.W && p.Y <= b.Y+b.H {
			if ed.selected != i {
				ed.point = -1
			}
			ed.selected = i
			e.drag = drag{on: true, start: p, handle: -1, shape: s.clone(), fit: ed.fit}
			return
		}
	}
	ed.selected, ed.point = -1, -1
	e.drag = drag{}
}

// pressSelected starts a drag on a handle of the selected shape, or on a
// point put on one of its polygon's edges; it reports whether p was on one.
func (e *tileEditor) pressSelected(p geom.Vec2, zoom float32, alt, twice bool) bool {
	ed := e.tool.ed
	s := ed.tile.Shapes[ed.selected]
	for i, h := range handles(s) {
		if h.Sub(p).Len()*zoom > 8 {
			continue
		}
		if s.Kind == Polygon {
			ed.point = i
			if alt {
				if err := ed.DeletePoint(); err != nil {
					e.tool.status.SetText(err.Error())
				}
				e.drag = drag{}
				return true
			}
		}
		e.drag = drag{on: true, start: p, handle: i, shape: s.clone(), fit: ed.fit}
		return true
	}
	if s.Kind != Polygon {
		return false
	}
	pts := handles(s)
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		at, ok := a.Add(b).Scale(0.5), false
		switch {
		case at.Sub(p).Len()*zoom <= 7:
			ok = true
		case twice && segmentDistance(p, a, b)*zoom <= 6:
			at, ok = p, true
		}
		if ok {
			ed.InsertPoint(i, at)
			s = ed.tile.Shapes[ed.selected]
			e.drag = drag{on: true, start: p, handle: ed.point, shape: s.clone(), fit: ed.fit}
			e.tool.changed()
			return true
		}
	}
	return false
}

// segmentDistance is how far p lies from the segment a, b.
func segmentDistance(p, a, b geom.Vec2) float32 {
	ab := b.Sub(a)
	t := float32(0)
	if l := ab.Dot(ab); l > 0 {
		t = min(max(p.Sub(a).Dot(ab)/l, 0), 1)
	}
	return p.Sub(a.Add(ab.Scale(t))).Len()
}

// commit ends a drag, as one change, if it changed anything.
func (e *tileEditor) commit(s Shape) {
	ed := e.tool.ed
	e.drag.on = false
	before := ed.tile.Shapes[ed.selected]
	if s.X == before.X && s.Y == before.Y && s.W == before.W && s.H == before.H &&
		pointsEqual(s.Points, before.Points) {
		return
	}
	t := ed.tile.clone()
	t.Shapes[ed.selected] = s
	sel := ed.selected
	ed.apply(t)
	ed.selected = sel
	e.tool.changed()
}

func pointsEqual(a, b []geom.Vec2) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ellipsePoints are 32 points round the ellipse in r.
func ellipsePoints(r geom.Rect) []geom.Vec2 {
	out := make([]geom.Vec2, 32)
	for i := range out {
		a := float64(i) / 32 * 2 * math.Pi
		out[i] = geom.V2(r.X+r.W/2+r.W/2*float32(math.Cos(a)), r.Y+r.H/2+r.H/2*float32(math.Sin(a)))
	}
	return out
}

// outline draws the closed path through pts.
func outline(c *twod.Canvas, pts []geom.Vec2, col color.Color) {
	for i := range pts {
		c.Line(pts[i], pts[(i+1)%len(pts)], 2, col)
	}
}

// previewPane is where the preview shows in the main window: the camera
// draws the lit tile and the walker there, under it; a click gives it the
// focus, and WASD walks the body while it has it.
type previewPane struct {
	slot
	tool    *Tool
	focused bool
}

// Focus gives the pane the keyboard, so WASD walks the body; a click
// elsewhere takes it back.
func (p *previewPane) Focus() { p.focused = true }

func (p *previewPane) Focused() bool { return p.focused }

func (p *previewPane) Draw(c *twod.Canvas) {
	size := c.Size()
	r := geom.Rect{W: size.X, H: size.Y}
	if c.Interact(r, "preview").Clicked {
		p.Focus()
	} else if in := c.Input(); in != nil && in.Pressed(input.MouseLeft) && !c.Hovered(r) {
		p.focused = false
	}
	col := color.Hex("#3a3d44")
	if p.Focused() {
		col = color.Hex("#ffd34a")
	}
	c.Border(r, 0, 1, col)
	if p.tool.ed.path == "" {
		hint := "Light and collision show here"
		c.Text(
			geom.V2(size.X/2-c.Measure(hint, twod.FontSize(14)).X/2, size.Y/2-8),
			hint,
			twod.FontSize(14),
			twod.Color(color.Hex("#8a8f99")),
		)
	}
	c.Record("preview", "canvas", "preview", p.Focused(), true, r, "press")
}

// command reports whether Command, on macOS, or Control, elsewhere, is
// down.
func command(in *input.State) bool {
	if runtime.GOOS == "darwin" {
		return in.Down(input.KeyLeftSuper) || in.Down(input.KeyRightSuper)
	}
	return in.Down(input.KeyLeftControl) || in.Down(input.KeyRightControl)
}

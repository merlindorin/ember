package light

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
)

// blockedCell is the side, in window points, of a cell of a blocked grid.
const blockedCell = 64

// blocked is where a window's occluders lie, a cell of blockedCell points
// a byte of the groups of those within it, so that a light none reaches
// casts no shadow at all.
type blocked struct {
	origin     geom.Vec2
	cols, rows int
	cells      []uint8
}

// reset empties b over the light rectangle r.
func (b *blocked) reset(r geom.Rect) {
	b.origin = geom.V2(r.X, r.Y)
	b.cols, b.rows = int(r.W/blockedCell)+1, int(r.H/blockedCell)+1
	b.cells = append(b.cells[:0], make([]uint8, b.cols*b.rows)...)
}

// span is the cells r, in points from the light rectangle's top-left,
// covers, clamped to the grid; ok is false when it covers none.
func (b *blocked) span(r geom.Rect) (int, int, int, int, bool) {
	var x0 int
	var y0 int
	var x1 int
	var y1 int
	x0, y0 = max(int(r.X/blockedCell), 0), max(int(r.Y/blockedCell), 0)
	x1, y1 = min(int((r.X+r.W)/blockedCell), b.cols-1), min(int((r.Y+r.H)/blockedCell), b.rows-1)
	return x0, y0, x1, y1, r.X+r.W >= 0 && r.Y+r.H >= 0 && x0 <= x1 && y0 <= y1
}

// add marks the cells r covers with groups.
func (b *blocked) add(r geom.Rect, groups uint8) {
	x0, y0, x1, y1, ok := b.span(r)
	if !ok {
		return
	}
	for y := y0; y <= y1; y++ {
		row := b.cells[y*b.cols:]
		for x := x0; x <= x1; x++ {
			row[x] |= groups
		}
	}
}

// any reports whether an occluder of groups may lie within r, in window
// points.
func (b *blocked) any(r geom.Rect, groups uint8) bool {
	x0, y0, x1, y1, ok := b.span(geom.Rect{X: r.X - b.origin.X, Y: r.Y - b.origin.Y, W: r.W, H: r.H})
	if !ok {
		return false
	}
	for y := y0; y <= y1; y++ {
		row := b.cells[y*b.cols:]
		for x := x0; x <= x1; x++ {
			if row[x]&groups != 0 {
				return true
			}
		}
	}
	return false
}

// unblock marks Unblocked each light of l no occluder of its groups reaches.
func (b *blocked) unblock(l *draw.Lighting) {
	for i := range l.Lights {
		li := &l.Lights[i]
		if li.Kind == draw.DirectionalLight || li.XRay {
			continue
		}
		reach := geom.Rect{X: li.At.X - li.Radius, Y: li.At.Y - li.Radius, W: 2 * li.Radius, H: 2 * li.Radius}
		if li.Kind == draw.ShapeLight {
			s := li.ShapeBounds()
			reach = geom.Rect{X: s.X - li.Radius, Y: s.Y - li.Radius, W: s.W + 2*li.Radius, H: s.H + 2*li.Radius}
		}
		li.Unblocked = !b.any(reach, li.Groups)
	}
}

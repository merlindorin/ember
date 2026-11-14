package paint

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
)

// HitTest is the element box at p, in the root's space, that paints on
// top there: the last one painted whose border box holds p, within its
// clips and through its transforms. Hidden boxes and text runs never are,
// but transparent ones are, as in CSS.
// Controls, if given, finds modal dialogs, whose backdrop takes the page's
// pointer.
func HitTest(t *box.Tree, p geom.Vec2, opts ...Option) *box.Box {
	pt := scan(t, opts, p, true)
	if pt == nil {
		return nil
	}
	defer scanners.Put(pt)
	hit := pt.found
	for _, pp := range pt.popups {
		if pt.place(pp).rect.Contains(p) {
			hit = pp.b
		}
	}
	return hit
}

// probed is told, on a hit test's walk, of each element box painted at r
// in the current space: the last holding point is the hit, but for one
// with pointer-events: none.
func (p *painter) probed(b *box.Box, r geom.Rect) {
	if st := shown(b); !p.pointless && (st == nil || !st.PointerNone) && p.probe.holds(p.point, r) {
		p.found = b
	}
}

// probe is a Surface that draws nothing, keeping only the transform and
// the clips, for hit testing.
type probe struct {
	xf    geom.Transform2D
	clips []probeClip
}

type probeClip struct {
	rect  geom.Rect
	radii [4]float32
	back  geom.Transform2D
}

// holds reports whether q, in the clip's space, is inside it: its rect,
// and with radii, its rounded corners.
func (c probeClip) holds(q geom.Vec2) bool {
	if !hits(c.rect, q) {
		return false
	}
	r := c.rect
	for i, corner := range cornersOf(r) {
		k := c.radii[i]
		if k <= 0 {
			continue
		}
		sx, sy := float32(1), float32(1)
		if i == 1 || i == 2 {
			sx = -1
		}
		if i >= 2 {
			sy = -1
		}
		centre := geom.V2(corner.X+sx*k, corner.Y+sy*k)
		d := q.Sub(centre)
		if d.X*sx < 0 && d.Y*sy < 0 && d.X*d.X+d.Y*d.Y > k*k {
			return false
		}
	}
	return true
}

// holds reports whether p, in the root's space, is in r in the current
// space and in every clip in theirs.
func (s *probe) holds(p geom.Vec2, r geom.Rect) bool {
	for _, c := range s.clips {
		if !c.holds(c.back.Xform(p)) {
			return false
		}
	}
	return hits(r, s.xf.Inverse().Xform(p))
}

// hits is whether p is in r as Chrome hits a box: its left and top
// edges in, its right and bottom ones out, so a point on a border between
// two boxes takes only the one starting there.
func hits(r geom.Rect, p geom.Vec2) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

func (s *probe) PushClip(r geom.Rect) {
	s.clips = append(s.clips, probeClip{rect: r, back: s.xf.Inverse()})
}

// PushClipRounded is a rounded clip, which is square under a rotation or
// a skew, as the draw list draws it (decision 0058).
func (s *probe) PushClipRounded(r geom.Rect, radii [4]float32) {
	if s.xf.X.Y != 0 || s.xf.Y.X != 0 {
		radii = [4]float32{}
	}
	s.clips = append(s.clips, probeClip{rect: r, radii: radii, back: s.xf.Inverse()})
}
func (s *probe) PopClip()                                               { s.clips = s.clips[:len(s.clips)-1] }
func (s *probe) Transform() geom.Transform2D                            { return s.xf }
func (s *probe) SetTransform(t geom.Transform2D)                        { s.xf = t }
func (s *probe) Rect(geom.Rect, color.Color)                            {}
func (s *probe) RoundRect(geom.Rect, float32, color.Color)              {}
func (s *probe) Border(geom.Rect, float32, float32, color.Color)        {}
func (s *probe) Glow(geom.Rect, float32, float32, color.Color)          {}
func (s *probe) Image(geom.Rect, *twod.Texture, geom.Rect, color.Color) {}
func (s *probe) NinePatch(geom.Rect, *twod.Texture, float32, float32, float32, float32, color.Color) {
}
func (s *probe) Line(geom.Vec2, geom.Vec2, float32, color.Color)           {}
func (s *probe) Circle(geom.Vec2, float32, color.Color)                    {}
func (s *probe) RoundRectCorners(geom.Rect, [4]float32, color.Color)       {}
func (s *probe) BorderCorners(geom.Rect, [4]float32, float32, color.Color) {}
func (s *probe) TrianglesColored([]geom.Vec2, []color.Color, []int)        {}
func (s *probe) Triangles([]geom.Vec2, []int, color.Color)                 {}
func (s *probe) Text(geom.Vec2, string, ...twod.TextOption) geom.Vec2      { return geom.Vec2{} }

// cornersOf are r's corners clockwise from the top-left, as radii and
// clips number them.
func cornersOf(r geom.Rect) [4]geom.Vec2 {
	return [4]geom.Vec2{{X: r.X, Y: r.Y}, {X: r.X + r.W, Y: r.Y}, {X: r.X + r.W, Y: r.Y + r.H}, {X: r.X, Y: r.Y + r.H}}
}

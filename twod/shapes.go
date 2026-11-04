package twod

import (
	"math"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/node"
)

//nolint:gochecknoinits // registers the package's node types on import
func init() {
	node.Register(node.Type{New: func() node.Noded { return NewLine(nil) }})
	node.Register(node.Type{New: func() node.Noded { return NewPolygon(nil) }})
}

// Joint is how a line turns at its points.
type Joint int

const (
	Miter Joint = iota
	Bevel
	RoundJoint
)

//nolint:gochecknoglobals // word table
var joints = node.Words[Joint]{"miter", "bevel", "round"}

func (v Joint) MarshalText() ([]byte, error)  { return joints.Text(v) }
func (v *Joint) UnmarshalText(b []byte) error { return joints.Parse(v, b) }

// Cap is how a line ends.
type Cap int

const (
	Flat Cap = iota
	Square
	RoundCap
)

//nolint:gochecknoglobals // word table
var caps = node.Words[Cap]{"flat", "square", "round"}

func (v Cap) MarshalText() ([]byte, error)  { return caps.Text(v) }
func (v *Cap) UnmarshalText(b []byte) error { return caps.Parse(v, b) }

// Line draws a polyline through its points.
type Line struct {
	Item
	points []geom.Vec2 `node:"prop"`
	width  float32     `node:"prop"`
	color  color.Color `node:"prop"`
	closed bool        `node:"prop"`
	joint  Joint       `node:"prop"`
	start  Cap         `node:"prop"`
	end    Cap         `node:"prop"`
}

// NewLine returns a white line 2 points wide through points.
func NewLine(points []geom.Vec2, opts ...LineOption) *Line {
	l := Build(&Line{Item: Base(), points: points, width: 2, color: color.White}, "line")
	for _, o := range opts {
		o.line(l)
	}
	return l
}

// Width is the line's thickness in points.
func Width(w float32) LineOption { return lineOption(func(l *Line) { l.width = w }) }

// Closed joins a line's last point back to its first.
func Closed() LineOption { return lineOption(func(l *Line) { l.closed = true }) }

// Joints sets how a line turns at its points (default Miter).
func Joints(j Joint) LineOption { return lineOption(func(l *Line) { l.joint = j }) }

// Caps sets how a line starts and ends (default Flat).
func Caps(start, end Cap) LineOption {
	return lineOption(func(l *Line) { l.start, l.end = start, end })
}

func (l *Line) Points() []geom.Vec2     { return l.points }
func (l *Line) SetPoints(p []geom.Vec2) { l.points = p }
func (l *Line) Color() color.Color      { return l.color }
func (l *Line) SetColor(c color.Color)  { l.color = c }
func (l *Line) Width() float32          { return l.width }
func (l *Line) SetWidth(w float32)      { l.width = w }

const (
	miterLimit     = 2
	roundPrecision = 8
)

type mesh struct {
	pts []geom.Vec2
	idx []int
}

func (m *mesh) tri(a, b, c geom.Vec2) {
	n := len(m.pts)
	m.pts = append(m.pts, a, b, c)
	m.idx = append(m.idx, n, n+1, n+2)
}

// arc fans around center from a to b by the shorter way.
func (m *mesh) arc(center, a, b geom.Vec2) {
	r := a.Sub(center).Len()
	from := math.Atan2(float64(a.Y-center.Y), float64(a.X-center.X))
	to := math.Atan2(float64(b.Y-center.Y), float64(b.X-center.X))
	d := to - from
	for d > math.Pi {
		d -= 2 * math.Pi
	}
	for d < -math.Pi {
		d += 2 * math.Pi
	}
	steps := max(int(math.Ceil(math.Abs(d)/(math.Pi/roundPrecision))), 1)
	prev := a
	for i := 1; i <= steps; i++ {
		ang := from + d*float64(i)/float64(steps)
		p := center.Add(geom.V2(float32(math.Cos(ang))*r, float32(math.Sin(ang))*r))
		m.tri(center, prev, p)
		prev = p
	}
}

func normal(a, b geom.Vec2, half float32) (geom.Vec2, geom.Vec2) {
	d := b.Sub(a).Normalize()
	return d, geom.V2(-d.Y, d.X).Scale(half)
}

// mesh is the line's triangles in its own space.
func (l *Line) mesh() ([]geom.Vec2, []int) {
	var m mesh
	pts := l.points
	if len(pts) < 2 {
		return nil, nil
	}
	half := l.width / 2
	segs := len(pts) - 1
	if l.closed {
		segs = len(pts)
	}
	at := func(i int) geom.Vec2 { return pts[i%len(pts)] }
	for i := range segs {
		a, b := at(i), at(i+1)
		d, n := normal(a, b, half)
		if !l.closed && i == 0 && l.start == Square {
			a = a.Sub(d.Scale(half))
		}
		if !l.closed && i == segs-1 && l.end == Square {
			b = b.Add(d.Scale(half))
		}
		m.tri(a.Add(n), b.Add(n), b.Sub(n))
		m.tri(a.Add(n), b.Sub(n), a.Sub(n))
	}
	turns := make([]int, 0, len(pts))
	for i := 1; i < len(pts)-1; i++ {
		turns = append(turns, i)
	}
	if l.closed {
		turns = append(turns, len(pts)-1, 0)
	}
	for _, i := range turns {
		p := pts[i]
		prev, next := pts[(i-1+len(pts))%len(pts)], pts[(i+1)%len(pts)]
		_, n1 := normal(prev, p, half)
		_, n2 := normal(p, next, half)
		for _, side := range []float32{1, -1} {
			m.joint(l.joint, p, n1.Scale(side), n2.Scale(side), half)
		}
	}
	if !l.closed {
		l.roundCaps(&m, half)
	}
	return m.pts, m.idx
}

// joint fills the turn at p between the edges offset by s1 and s2, half the
// line's width out.
func (m *mesh) joint(j Joint, p, s1, s2 geom.Vec2, half float32) {
	switch j {
	case RoundJoint:
		m.arc(p, p.Add(s1), p.Add(s2))
	case Miter:
		bis := s1.Add(s2)
		cos := bis.Len() / (2 * half)
		if cos > 1e-4 && 1/cos <= miterLimit {
			tip := p.Add(bis.Normalize().Scale(half / cos))
			m.tri(p, p.Add(s1), tip)
			m.tri(p, tip, p.Add(s2))
			return
		}
		m.tri(p, p.Add(s1), p.Add(s2))
	case Bevel:
		m.tri(p, p.Add(s1), p.Add(s2))
	}
}

// roundCaps adds the half discs of an open line's RoundCap ends.
func (l *Line) roundCaps(m *mesh, half float32) {
	pts := l.points
	if l.start == RoundCap {
		_, n := normal(pts[0], pts[1], half)
		d, _ := normal(pts[1], pts[0], half)
		p := pts[0]
		m.arc(p, p.Add(n), p.Add(d.Scale(half)))
		m.arc(p, p.Add(d.Scale(half)), p.Sub(n))
	}
	if l.end == RoundCap {
		last := len(pts) - 1
		d, n := normal(pts[last-1], pts[last], half)
		p := pts[last]
		m.arc(p, p.Add(n), p.Add(d.Scale(half)))
		m.arc(p, p.Add(d.Scale(half)), p.Sub(n))
	}
}

func (l *Line) Draw(c *Canvas) {
	if pts, idx := l.mesh(); len(idx) > 0 {
		c.Triangles(pts, idx, l.color)
	}
}

// Polygon fills a shape; concave shapes are triangulated.
type Polygon struct {
	Item
	points []geom.Vec2 `node:"prop"`
	color  color.Color `node:"prop"`
}

// NewPolygon returns a white polygon through points.
func NewPolygon(points []geom.Vec2, opts ...PolygonOption) *Polygon {
	p := Build(&Polygon{Item: Base(), points: points, color: color.White}, "polygon")
	for _, o := range opts {
		o.polygon(p)
	}
	return p
}

func (p *Polygon) Points() []geom.Vec2     { return p.points }
func (p *Polygon) SetPoints(q []geom.Vec2) { p.points = q }
func (p *Polygon) Color() color.Color      { return p.color }
func (p *Polygon) SetColor(c color.Color)  { p.color = c }

func (p *Polygon) Draw(c *Canvas) {
	if idx := triangulate(p.points); len(idx) > 0 {
		c.Triangles(p.points, idx, p.color)
	}
}

// triangulate splits a simple polygon of either winding into triangles by
// ear clipping, returning indices into it.
func triangulate(poly []geom.Vec2) []int {
	n := len(poly)
	if n < 3 {
		return nil
	}
	var signed float32
	for i := range n {
		a, b := poly[i], poly[(i+1)%n]
		signed += a.X*b.Y - b.X*a.Y
	}
	left := make([]int, n)
	for i := range left {
		left[i] = i
	}
	if signed < 0 {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			left[i], left[j] = left[j], left[i]
		}
	}
	cross := func(a, b, c geom.Vec2) float32 { return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X) }
	inside := func(p, a, b, c geom.Vec2) bool {
		return cross(a, b, p) >= 0 && cross(b, c, p) >= 0 && cross(c, a, p) >= 0
	}
	var out []int
	for guard := 0; len(left) > 3 && guard < n*n; guard++ {
		clipped := false
		for i := range left {
			ia, ib, ic := left[(i+len(left)-1)%len(left)], left[i], left[(i+1)%len(left)]
			a, b, c := poly[ia], poly[ib], poly[ic]
			if cross(a, b, c) <= 0 {
				continue
			}
			ear := true
			for _, j := range left {
				if j != ia && j != ib && j != ic && inside(poly[j], a, b, c) {
					ear = false
					break
				}
			}
			if ear {
				out = append(out, ia, ib, ic)
				left = append(left[:i], left[i+1:]...)
				clipped = true
				break
			}
		}
		if !clipped {
			break
		}
	}
	if len(left) == 3 {
		out = append(out, left...)
	}
	return out
}

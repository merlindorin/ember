package paint

import (
	"math"
	"sync"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// gradientSteps is how many bands an interval between two stops is cut
// into. The GPU blends a vertex's colour in linear light, while Chrome
// blends a gradient's in encoded sRGB (decision 0056); bands this narrow
// follow its curve within a level or two.
const gradientSteps = 12

// sample is a colour on the gradient line, at u from its start, in
// encoded sRGB, premultiplied.
type sample struct {
	u    float32
	rgba [4]float32
}

// gradient fills area, the padding box, rounded by radius, with g: the
// box cut into bands along the gradient line, each band's colour blended
// between its two edges, the line's ends held past the first and the
// last stop. Rounded, the bands fill the whole box inside a rounded clip
// (decision 0058), whose coverage anti-aliases the corners as Chrome does;
// inside a rounded overflow clip, that one is then the innermost, and the
// outer curve cuts the gradient as its rect.
func (p *painter) gradient(area geom.Rect, radius radii, g *style.Gradient, alpha float32) {
	if len(g.Stops) == 0 || area.W <= 0 || area.H <= 0 {
		return
	}
	if radius != (radii{}) {
		p.s.PushClipRounded(area, radius)
		defer p.s.PopClip()
		radius = radii{}
	}
	switch g.Kind {
	case style.ConicGradient:
		p.s.PushClip(area)
		p.conic(area, g, alpha)
		p.s.PopClip()
		return
	case style.RadialGradient:
		p.s.PushClip(area)
		p.radial(area, g, alpha)
		p.s.PopClip()
		return
	case style.LinearGradient:
	}
	d := lineDirection(g, area)
	span := float32(math.Abs(float64(area.W*d.X)) + math.Abs(float64(area.H*d.Y)))
	start := geom.V2(area.X+area.W/2, area.Y+area.H/2).Sub(d.Scale(span / 2))
	along := func(q geom.Vec2) float32 { return q.Sub(start).Dot(d) }
	samples := gradientSamples(g, span)
	shape := roundedShape(area, radius)
	var points []geom.Vec2
	var colors []color.Color
	var indices []int
	piece := func(poly []geom.Vec2, at func(u float32) color.Color) {
		if len(poly) < 3 {
			return
		}
		first := len(points)
		for _, q := range poly {
			points = append(points, q)
			colors = append(colors, fade(at(along(q)), alpha))
		}
		for i := 1; i+1 < len(poly); i++ {
			indices = append(indices, first, first+i, first+i+1)
		}
	}
	first, last := samples[0], samples[len(samples)-1]
	piece(cut(shape, along, float32(math.Inf(-1)), first.u), func(float32) color.Color { return decoded(first.rgba) })
	for i := 0; i+1 < len(samples); i++ {
		a, b := samples[i], samples[i+1]
		if b.u <= a.u {
			continue
		}
		ca, cb := decoded(a.rgba), decoded(b.rgba)
		piece(cut(shape, along, a.u, b.u), func(u float32) color.Color { return ca.Lerp(cb, (u-a.u)/(b.u-a.u)) })
	}
	piece(cut(shape, along, last.u, float32(math.Inf(1))), func(float32) color.Color { return decoded(last.rgba) })
	if len(indices) > 0 {
		p.s.TrianglesColored(points, colors, indices)
	}
}

// lineDirection is the gradient line's unit direction in area: its angle,
// 0 up and clockwise, or, toward a corner, square to the diagonal through
// the two corners beside it.
func lineDirection(g *style.Gradient, area geom.Rect) geom.Vec2 {
	if g.Corner != ([2]int8{}) {
		return geom.V2(float32(g.Corner[0])*area.H, float32(g.Corner[1])*area.W).Normalize()
	}
	a := float64(g.Angle) * math.Pi / 180
	return geom.V2(float32(math.Sin(a)), float32(-math.Cos(a)))
}

// gradientSamples are g's stops on a line span long, placed as CSS places
// them, each interval between two of a different colour cut into
// gradientSteps.
func gradientSamples(g *style.Gradient, span float32) []sample {
	n := len(g.Stops)
	at := make([]float32, n)
	placed := make([]bool, n)
	most := float32(math.Inf(-1))
	for i, s := range g.Stops {
		switch {
		case s.Placed:
			at[i], placed[i] = s.At.Px+s.At.Percent*span/100, true
		case i == 0:
			at[i], placed[i] = 0, true
		case i == n-1:
			at[i], placed[i] = span, true
		}
		if placed[i] {
			at[i] = max(at[i], most)
			most = at[i]
		}
	}
	for i := 0; i < n; i++ {
		if placed[i] {
			continue
		}
		j := i
		for !placed[j] {
			j++
		}
		from, to := at[i-1], at[j]
		for k := i; k < j; k++ {
			at[k] = from + (to-from)*float32(k-i+1)/float32(j-i+1)
		}
		i = j
	}
	out := make([]sample, 0, n*gradientSteps)
	for i, s := range g.Stops {
		c := encoded(s.Color)
		if i > 0 {
			prev := out[len(out)-1]
			if steps := gradientSteps; prev.rgba != c && at[i] > prev.u {
				for k := 1; k < steps; k++ {
					t := float32(k) / float32(steps)
					out = append(out, sample{prev.u + (at[i]-prev.u)*t, mix(prev.rgba, c, t)})
				}
			}
		}
		out = append(out, sample{at[i], c})
	}
	return out
}

// encoded is c in encoded sRGB, premultiplied: what Chrome blends.
func encoded(c color.Color) [4]float32 {
	e := c.SRGB()
	return [4]float32{e[0] * c.A, e[1] * c.A, e[2] * c.A, c.A}
}

// decoded is a premultiplied encoded colour back in linear light.
func decoded(e [4]float32) color.Color {
	if e[3] <= 0 {
		return color.Color{}
	}
	c := color.FromSRGB([3]float32{e[0] / e[3], e[1] / e[3], e[2] / e[3]})
	c.A = e[3]
	return c
}

func mix(a, b [4]float32, t float32) [4]float32 {
	for i := range a {
		a[i] += (b[i] - a[i]) * t
	}
	return a
}

// cut is the part of the convex polygon poly whose along lies within
// [lo, hi].
func cut(poly []geom.Vec2, along func(geom.Vec2) float32, lo, hi float32) []geom.Vec2 {
	poly = keep(poly, func(q geom.Vec2) float32 { return along(q) - lo })
	return keep(poly, func(q geom.Vec2) float32 { return hi - along(q) })
}

// keep is the part of poly where side is at least 0.
func keep(poly []geom.Vec2, side func(geom.Vec2) float32) []geom.Vec2 {
	var out []geom.Vec2
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		sa, sb := side(a), side(b)
		if sa >= 0 {
			out = append(out, a)
		}
		if (sa >= 0) != (sb >= 0) && !math.IsInf(float64(sa-sb), 0) {
			out = append(out, a.Add(b.Sub(a).Scale(sa/(sa-sb))))
		}
	}
	return out
}

// roundedShape is r with its corners rounded by radius, as a convex
// polygon, clockwise from the top-left arc.
func roundedShape(r geom.Rect, radius radii) []geom.Vec2 {
	corners := [4]struct {
		at    geom.Vec2
		from  float64
		round float32
	}{
		{geom.V2(r.X+radius[0], r.Y+radius[0]), math.Pi, radius[0]},
		{geom.V2(r.X+r.W-radius[1], r.Y+radius[1]), 1.5 * math.Pi, radius[1]},
		{geom.V2(r.X+r.W-radius[2], r.Y+r.H-radius[2]), 0, radius[2]},
		{geom.V2(r.X+radius[3], r.Y+r.H-radius[3]), 0.5 * math.Pi, radius[3]},
	}
	var out []geom.Vec2
	for _, c := range corners {
		if c.round <= 0 {
			out = append(out, c.at)
			continue
		}
		steps := int(min(max(math.Ceil(float64(c.round)/2), 3), 16))
		for k := 0; k <= steps; k++ {
			a := c.from + float64(k)/float64(steps)*math.Pi/2
			out = append(out, c.at.Add(geom.V2(float32(math.Cos(a)), float32(math.Sin(a))).Scale(c.round)))
		}
	}
	return out
}

// band is a stretch of a conic or radial gradient, from u0 to u1, its
// colour blended from c0 to c1.
type band struct {
	u0, u1 float32
	c0, c1 color.Color
}

// bands are samples as the stretches between them, from 0 to end: the
// first stop's colour before it, the last's after it.
func bands(samples []sample, end float32) []band {
	first, last := samples[0], samples[len(samples)-1]
	out := []band{{0, first.u, decoded(first.rgba), decoded(first.rgba)}}
	for i := 0; i+1 < len(samples); i++ {
		a, b := samples[i], samples[i+1]
		if b.u > a.u {
			out = append(out, band{a.u, b.u, decoded(a.rgba), decoded(b.rgba)})
		}
	}
	return append(out, band{last.u, max(end, last.u), decoded(last.rgba), decoded(last.rgba)})
}

// centreOf is a conic or radial gradient's centre in area.
func centreOf(g *style.Gradient, area geom.Rect) geom.Vec2 {
	return geom.V2(place(g.At.X, g.At.FromRight, area.X, area.W, 0), place(g.At.Y, g.At.FromBottom, area.Y, area.H, 0))
}

// conicSteps is how many triangles a whole turn is cut into at least.
const conicSteps = 96

// conic fills area with the conic gradient g: a fan from its centre, a
// turn from its from angle, 0 up and clockwise, each band's colour blended
// across it, wide enough to reach every corner of area.
func (p *painter) conic(area geom.Rect, g *style.Gradient, alpha float32) {
	c := centreOf(g, area)
	reach := float32(1)
	for _, q := range cornersOf(area) {
		reach = max(reach, q.Sub(c).Len()*1.5)
	}
	at := func(u float32) geom.Vec2 {
		a := float64(g.Angle)*math.Pi/180 + float64(u)*2*math.Pi
		return c.Add(geom.V2(float32(math.Sin(a)), float32(-math.Cos(a))).Scale(reach))
	}
	var points []geom.Vec2
	var colors []color.Color
	var indices []int
	for _, b := range bands(gradientSamples(g, 1), 1) {
		n := max(int(math.Ceil(float64((b.u1-b.u0)*conicSteps))), 1)
		for k := range n {
			t0, t1 := float32(k)/float32(n), float32(k+1)/float32(n)
			u0, u1 := b.u0+(b.u1-b.u0)*t0, b.u0+(b.u1-b.u0)*t1
			first := len(points)
			points = append(points, c, at(u0), at(u1))
			mid, start, end := b.c0.Lerp(b.c1, (t0+t1)/2), b.c0.Lerp(b.c1, t0), b.c0.Lerp(b.c1, t1)
			colors = append(colors, fade(mid, alpha), fade(start, alpha), fade(end, alpha))
			indices = append(indices, first, first+1, first+2)
		}
	}
	if len(indices) > 0 {
		p.s.TrianglesColored(points, colors, indices)
	}
}

// radialSteps is how many sides a radial gradient's ellipses are drawn
// with.
const radialSteps = 96

// radialRadii are a radial gradient's radii across and down in area, as
// CSS sizes it: to its closest or farthest side, or through its closest
// or farthest corner, an ellipse keeping its sides' proportions.
func radialRadii(g *style.Gradient, area geom.Rect, c geom.Vec2) (float32, float32) {
	l, r := abs32(c.X-area.X), abs32(area.X+area.W-c.X)
	t, b := abs32(c.Y-area.Y), abs32(area.Y+area.H-c.Y)
	near, far := geom.V2(min(l, r), min(t, b)), geom.V2(max(l, r), max(t, b))
	var rx, ry float32
	switch g.Size {
	case style.ClosestSide:
		rx, ry = near.X, near.Y
		if g.Circle {
			rx = min(near.X, near.Y)
			ry = rx
		}
	case style.FarthestSide:
		rx, ry = far.X, far.Y
		if g.Circle {
			rx = max(far.X, far.Y)
			ry = rx
		}
	case style.ClosestCorner:
		rx, ry = near.X*math.Sqrt2, near.Y*math.Sqrt2
		if g.Circle {
			rx = near.Len()
			ry = rx
		}
	case style.FarthestCorner:
		rx, ry = far.X*math.Sqrt2, far.Y*math.Sqrt2
		if g.Circle {
			rx = far.Len()
			ry = rx
		}
	}
	return max(rx, 1e-3), max(ry, 1e-3)
}

func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }

// radial fills area with the radial gradient g: rings between ellipses
// round its centre, each band's colour blended from its inner edge to its
// outer, the last colour reaching every corner of area. Stops in lengths
// count along the ellipse's horizontal radius, as in CSS.
func (p *painter) radial(area geom.Rect, g *style.Gradient, alpha float32) {
	c := centreOf(g, area)
	rx, ry := radialRadii(g, area, c)
	reach := float32(1)
	for _, q := range cornersOf(area) {
		d := q.Sub(c)
		reach = max(reach, float32(math.Hypot(float64(d.X/rx), float64(d.Y/ry)))*1.05)
	}
	samples := gradientSamples(g, rx)
	for i := range samples {
		samples[i].u /= rx
	}
	m := pooled[mesh](&meshes)
	defer meshes.Put(m)
	m.reset()
	ring := func(f float32, col color.Color) int {
		first := len(m.points)
		for _, t := range radialTurn {
			m.points = append(m.points, c.Add(geom.V2(t.X*rx*f, t.Y*ry*f)))
			m.colors = append(m.colors, col)
		}
		return first
	}
	for _, b := range bands(samples, reach) {
		if b.u1 <= b.u0 {
			continue
		}
		c0, c1 := fade(b.c0, alpha), fade(b.c1, alpha)
		out := ring(b.u1, c1)
		if b.u0 == 0 {
			mid := len(m.points)
			m.points, m.colors = append(m.points, c), append(m.colors, c0)
			for k := range radialSteps {
				m.indices = append(m.indices, mid, out+k, out+(k+1)%radialSteps)
			}
			continue
		}
		in := ring(b.u0, c0)
		for k := range radialSteps {
			k1 := (k + 1) % radialSteps
			m.indices = append(m.indices, in+k, in+k1, out+k1, in+k, out+k1, out+k)
		}
	}
	if len(m.indices) > 0 {
		p.s.TrianglesColored(m.points, m.colors, m.indices)
	}
}

// meshes are gradients' meshes, reused from one paint to the next.
//
//nolint:gochecknoglobals // a sync.Pool every paint shares
var meshes = sync.Pool{New: func() any { return new(mesh) }}

func (m *mesh) reset() {
	m.points, m.colors, m.indices = m.points[:0], m.colors[:0], m.indices[:0]
}

// radialTurn are the points of a unit circle radialSteps cut, clockwise
// from the right.
//
//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var radialTurn = func() [radialSteps]geom.Vec2 {
	var out [radialSteps]geom.Vec2
	for k := range out {
		a := 2 * math.Pi * float64(k) / radialSteps
		out[k] = geom.V2(float32(math.Cos(a)), float32(math.Sin(a)))
	}
	return out
}()

package paint

import (
	"math"
	"slices"
	"sync"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var whole = geom.Rect{W: 1, H: 1}

// piece is one part of an element's border box: the whole box, or one
// line's fragment of a split inline, with the border edges it carries.
type piece struct {
	rect  geom.Rect
	edges box.Edges
}

// pieces are b's parts in the root's space, b's border box at at: one per
// line of an inline split over several, the left edge on the first and
// the right edge on the last, else the box itself.
// They go in p's scratch, valid until the next call.
func (p *painter) pieces(b *box.Box, at geom.Vec2) []piece {
	if len(b.Fragments) <= 1 {
		s := framed(b)
		p.scratch = append(p.scratch[:0], piece{geom.Rect{X: at.X, Y: at.Y + b.Wrap.Top, W: s.X, H: s.Y}, b.Border})
		return p.scratch
	}
	out := slices.Grow(p.scratch[:0], len(b.Fragments))[:len(b.Fragments)]
	p.scratch = out
	for i, f := range b.Fragments {
		e := b.Border
		if i > 0 {
			e.Left = 0
		}
		if i < len(b.Fragments)-1 {
			e.Right = 0
		}
		out[i] = piece{geom.Rect{X: at.X + f.X, Y: at.Y + f.Y, W: f.W, H: f.H}, e}
	}
	return out
}

// rowBackground paints a table row's background behind each of its own
// cells, over a cell's whole box when it spans rows, as Chrome paints
// rows (CSS 2.1 §17.5.1): a row below never paints over a cell spanning
// into it from above, and the gaps border-spacing leaves show what's
// under the row.
func (p *painter) rowBackground(row *box.Box, at geom.Vec2, c color.Color) {
	for _, cell := range row.Children {
		if cell.Kind != box.Element || cell.Node == nil || (cell.Node.Tag() != dom.Td && cell.Node.Tag() != dom.Th) {
			continue
		}
		p.fill(p.snap(geom.Rect{X: at.X + cell.Pos.X, Y: at.Y + cell.Pos.Y, W: cell.Size.X, H: cell.Size.Y}), radii{}, c)
	}
}

// background paints e's background colour, its row's for a table row,
// then its gradient or image inside edges, over r rounded by radius.
func (p *painter) background(e entry, r geom.Rect, edges box.Edges, radius radii, st *style.Computed, alpha float32) {
	b := e.b
	if st.Background.A > 0 && !tracked(b) {
		if b.Node != nil && b.Node.Tag() == dom.Tr {
			p.rowBackground(b, e.at, fade(st.Background, alpha))
		} else {
			p.fill(r, radius, fade(st.Background, alpha))
		}
	}
	if g := st.BackgroundImage.Gradient; g != nil {
		p.gradient(inside(r, edges), radius.inside(edges), g, alpha)
	} else {
		p.backgroundImage(inside(r, edges), st, alpha)
	}
}

// decorate paints an element's shadows, background and border, on each
// of its pieces; a split inline's are square.
func (p *painter) decorate(e entry, alpha float32) {
	b, st := e.b, shown(e.b)
	if b.Kind != box.Element || !st.Visible || p.culled(e, inkOf(e, st)) {
		return
	}
	ps := p.pieces(b, e.at)
	if p.probing {
		for _, pc := range ps {
			p.probed(b, pc.rect)
		}
		return
	}
	radius := radiiOf(st, framed(b))
	if len(ps) > 1 {
		radius = radii{}
	}
	for _, pc := range ps {
		r := p.snap(pc.rect)
		for i := len(st.Shadows) - 1; i >= 0; i-- {
			if sh := st.Shadows[i]; !sh.Inset {
				p.shadow(r, radius, sh, alpha)
			}
		}
		if b != p.canvas {
			p.background(e, r, pc.edges, radius, st, alpha)
		}
		for i := len(st.Shadows) - 1; i >= 0; i-- {
			if sh := st.Shadows[i]; sh.Inset {
				p.insetShadow(inside(r, pc.edges), radius.inside(pc.edges), sh, alpha)
			}
		}
		p.border(r, pc.edges, radius, st, alpha)
		if tex := p.find(st.BorderImage, st); tex != nil {
			p.picture(r, tex, st.BorderImage.Slice, alpha)
		}
	}
	p.selection(e, alpha)
}

// find is img's texture through Images, its pixelated copy when st's
// image-rendering says so.
func (p *painter) find(img style.Image, st *style.Computed) *twod.Texture {
	if img.Source == "" || p.image == nil {
		return nil
	}
	tex := p.image(img.Source)
	if tex == nil || st.ImageRendering != style.RenderPixelated {
		return tex
	}
	return pixelated(tex)
}

// inside is r less edges: a border box's padding box.
func inside(r geom.Rect, e box.Edges) geom.Rect {
	return geom.Rect{X: r.X + e.Left, Y: r.Y + e.Top, W: r.W - e.Horizontal(), H: r.H - e.Vertical()}
}

// maxTiles bounds how many copies of a repeated background image are
// drawn, so a tiny image over a big box can't flood the draw list.
const maxTiles = 4096

// backgroundImage draws st's background image at its background-size,
// else its own size, in area, the padding box: placed by
// background-position, tiled from there both ways unless no-repeat,
// clipped to area.
func (p *painter) backgroundImage(area geom.Rect, st *style.Computed, alpha float32) {
	tex := p.find(st.BackgroundImage, st)
	if tex == nil {
		return
	}
	size := tex.Size()
	if size.X <= 0 || size.Y <= 0 || area.W <= 0 || area.H <= 0 {
		return
	}
	switch w, h := st.BackgroundSize[0], st.BackgroundSize[1]; {
	case w > 0 && h > 0:
		size = geom.V2(w, h)
	case w > 0:
		size = geom.V2(w, size.Y*w/size.X)
	case h > 0:
		size = geom.V2(size.X*h/size.Y, h)
	}
	pos := st.BackgroundPosition
	x := place(pos.X, pos.FromRight, area.X, area.W, size.X)
	y := place(pos.Y, pos.FromBottom, area.Y, area.H, size.Y)
	tint := fade(color.White, alpha)
	p.s.PushClip(area)
	defer p.s.PopClip()
	if st.BackgroundNoRepeat {
		p.s.Image(geom.Rect{X: x, Y: y, W: size.X, H: size.Y}, tex, whole, tint)
		return
	}
	x0 := x - float32(math.Ceil(float64((x-area.X)/size.X)))*size.X
	y0 := y - float32(math.Ceil(float64((y-area.Y)/size.Y)))*size.Y
	n := 0
	for ty := y0; ty < area.Y+area.H; ty += size.Y {
		for tx := x0; tx < area.X+area.W; tx += size.X {
			if n++; n > maxTiles {
				return
			}
			p.s.Image(geom.Rect{X: tx, Y: ty, W: size.X, H: size.Y}, tex, whole, tint)
		}
	}
}

// place is where an image of length img starts along an area from origin
// of length extent, at l from the start, or from the end when fromEnd: a
// percentage puts the image's point at the area's point.
func place(l style.Length, fromEnd bool, origin, extent, img float32) float32 {
	off := l.Px + l.Percent*(extent-img)/100
	if fromEnd {
		return origin + extent - img - off
	}
	return origin + off
}

// picture stretches tex over r, keeping slice unstretched when set.
func (p *painter) picture(r geom.Rect, tex *twod.Texture, slice style.Sides[float32], alpha float32) {
	tint := fade(color.White, alpha)
	if slice == (style.Sides[float32]{}) {
		p.s.Image(r, tex, whole, tint)
		return
	}
	p.ninePatch(r, tex, slice, tint)
}

func (p *painter) fill(r geom.Rect, radius radii, c color.Color) {
	switch {
	case radius.zero():
		p.s.Rect(r, c)
	case radius.uniform():
		p.s.RoundRect(r, radius[0], c)
	default:
		p.s.RoundRectCorners(r, radius, c)
	}
}

// shadow paints an outer box-shadow: the box moved by its offset and
// grown by its spread, sharp, or blurred as CSS blurs it, a Gaussian of
// sigma half the blur on a mesh three sigmas round it; either way
// clipped to outside the border box r, as CSS draws it, so a
// translucent box never shows its own shadow.
func (p *painter) shadow(r geom.Rect, radius radii, sh style.Shadow, alpha float32) {
	c := fade(sh.Color, alpha)
	if c.A <= 0 {
		return
	}
	s := geom.Rect{X: r.X + sh.X - sh.Spread, Y: r.Y + sh.Y - sh.Spread, W: r.W + 2*sh.Spread, H: r.H + 2*sh.Spread}
	grown := radius.grown(sh.Spread)
	if sh.Blur <= 0 {
		if radius.zero() {
			for _, part := range outside(s, r) {
				p.s.Rect(part, c)
			}
			return
		}
		p.fill(s, grown, c)
		return
	}
	m := shadowMesh(shadowKey{
		size: geom.V2(s.W, s.H), rounding: grown, shadow: style.Shadow{Blur: sh.Blur}, color: c, outer: true,
		box: geom.Rect{X: r.X - s.X, Y: r.Y - s.Y, W: r.W, H: r.H}, boxRounding: radius,
	})
	xf := p.s.Transform()
	p.s.SetTransform(xf.Mul(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(s.X, s.Y)}))
	p.s.TrianglesColored(m.points, m.colors, m.indices)
	p.s.SetTransform(xf)
}

// outside is what of a lies outside b, in up to four rectangles: the
// bands above and below b, then those beside it.
func outside(a, b geom.Rect) []geom.Rect {
	ix0, iy0 := max(a.X, b.X), max(a.Y, b.Y)
	ix1, iy1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if ix1 <= ix0 || iy1 <= iy0 {
		return []geom.Rect{a}
	}
	var out []geom.Rect
	for _, r := range []geom.Rect{
		{X: a.X, Y: a.Y, W: a.W, H: iy0 - a.Y},
		{X: a.X, Y: iy1, W: a.W, H: a.Y + a.H - iy1},
		{X: a.X, Y: iy0, W: ix0 - a.X, H: iy1 - iy0},
		{X: ix1, Y: iy0, W: a.X + a.W - ix1, H: iy1 - iy0},
	} {
		if r.W > 0 && r.H > 0 {
			out = append(out, r)
		}
	}
	return out
}

// gridMesh is a mesh over the grid of xs by ys, each point c with alpha
// times its colour's.
func gridMesh(xs, ys []float32, c color.Color, alpha func(x, y float32) float32) *mesh {
	m := &mesh{points: make([]geom.Vec2, 0, len(xs)*len(ys)), colors: make([]color.Color, 0, len(xs)*len(ys))}
	for _, y := range ys {
		for _, x := range xs {
			k := c
			k.A = c.A * alpha(x, y)
			m.points, m.colors = append(m.points, geom.V2(x, y)), append(m.colors, k)
		}
	}
	n := len(xs)
	m.indices = make([]int, 0, 6*(len(xs)-1)*(len(ys)-1))
	blank := func(t ...int) bool {
		for _, i := range t {
			if m.colors[i].A > 0 {
				return false
			}
		}
		return true
	}
	for j := 0; j+1 < len(ys); j++ {
		for i := 0; i+1 < n; i++ {
			a := j*n + i
			if !blank(a, a+1, a+n+1) {
				m.indices = append(m.indices, a, a+1, a+n+1)
			}
			if !blank(a, a+n+1, a+n) {
				m.indices = append(m.indices, a, a+n+1, a+n)
			}
		}
	}
	m.compact()
	return m
}

// compact keeps only the points m's triangles use, in their order.
func (m *mesh) compact() {
	at := make([]int, len(m.points))
	for i := range at {
		at[i] = -1
	}
	var points []geom.Vec2
	var colors []color.Color
	for j, i := range m.indices {
		if at[i] < 0 {
			at[i] = len(points)
			points, colors = append(points, m.points[i]), append(colors, m.colors[i])
		}
		m.indices[j] = at[i]
	}
	m.points, m.colors = points, colors
}

// insetShadow paints an inset box-shadow inside the padding box pad: all
// of it but a hole, pad moved by the offset and shrunk by the spread. A
// blurred one is CSS's Gaussian blur of that shape (sigma half the blur),
// exact because a rectangle's blur separates: 1 - gx(x)·gy(y), drawn as a
// mesh dense round the hole's edges. A rounded box's goes through
// roundedInset.
func (p *painter) insetShadow(pad geom.Rect, rounding radii, sh style.Shadow, alpha float32) {
	c := fade(sh.Color, alpha)
	if c.A <= 0 || pad.W <= 0 || pad.H <= 0 {
		return
	}
	if !rounding.zero() {
		p.roundedInset(pad, rounding, sh, c)
		return
	}
	hole := geom.Rect{
		X: pad.X + sh.X + sh.Spread, Y: pad.Y + sh.Y + sh.Spread,
		W: pad.W - 2*sh.Spread, H: pad.H - 2*sh.Spread,
	}
	hole = grow(hole, 0)
	p.s.PushClip(pad)
	defer p.s.PopClip()
	sigma := max(sh.Blur, 0) / 2
	if sigma <= 0 {
		for _, band := range []geom.Rect{
			{X: pad.X, Y: pad.Y, W: pad.W, H: hole.Y - pad.Y},
			{X: pad.X, Y: hole.Y + hole.H, W: pad.W, H: pad.Y + pad.H - hole.Y - hole.H},
			{X: pad.X, Y: hole.Y, W: hole.X - pad.X, H: hole.H},
			{X: hole.X + hole.W, Y: hole.Y, W: pad.X + pad.W - hole.X - hole.W, H: hole.H},
		} {
			if band.W > 0 && band.H > 0 {
				p.s.Rect(band, c)
			}
		}
		return
	}
	xs := blurNodes(pad.X, pad.X+pad.W, hole.X, hole.X+hole.W, sigma)
	ys := blurNodes(pad.Y, pad.Y+pad.H, hole.Y, hole.Y+hole.H, sigma)
	m := gridMesh(xs, ys, c, func(x, y float32) float32 {
		return 1 - blurredInterval(x, hole.X, hole.X+hole.W, sigma)*blurredInterval(y, hole.Y, hole.Y+hole.H, sigma)
	})
	p.s.TrianglesColored(m.points, m.colors, m.indices)
}

// blurOffsets are where, in sigmas from an edge, a blurred edge is sampled.
//
//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var blurOffsets = []float32{-3, -2, -1.5, -1, -0.5, 0, 0.5, 1, 1.5, 2, 3}

// blurNodes are the mesh's coordinates from lo to hi along one axis:
// its ends, and samples round the hole's edges a and b.
func blurNodes(lo, hi, a, b, sigma float32) []float32 {
	out := []float32{lo, hi}
	for _, e := range []float32{a, b} {
		for _, t := range blurOffsets {
			if v := e + t*sigma; v > lo && v < hi {
				out = append(out, v)
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// blurredInterval is the Gaussian blur, by sigma, of the interval [a, b]
// seen at x: 1 well inside it, 0 well outside.
func blurredInterval(x, a, b, sigma float32) float32 {
	cdf := func(v float32) float64 { return 0.5 * math.Erfc(-float64(v)/(float64(sigma)*math.Sqrt2)) }
	return float32(cdf(x-a) - cdf(x-b))
}

// roundedInset paints an inset shadow in a rounded padding box: its hole
// rounded by the box's radii less the spread, each point shaded by the
// Gaussian of its distance to the hole, sigma half the blur (a quarter
// point at least, an antialiased edge where there is no blur), on a mesh
// dense round the hole's edges and through the corners, its cells
// clipped to the rounded box so nothing shows past the corners.
func (p *painter) roundedInset(pad geom.Rect, rounding radii, sh style.Shadow, c color.Color) {
	m := shadowMesh(shadowKey{size: geom.V2(pad.W, pad.H), rounding: rounding, shadow: sh, color: c})
	if len(m.indices) == 0 {
		return
	}
	xf := p.s.Transform()
	p.s.SetTransform(xf.Mul(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(pad.X, pad.Y)}))
	p.s.TrianglesColored(m.points, m.colors, m.indices)
	p.s.SetTransform(xf)
}

// shadowKey is what a blurred shadow's mesh depends on: its box's size,
// not where it is, so a moved or scrolled box keeps its mesh; outer for
// an outer shadow, whose box is the spread one.
type shadowKey struct {
	size     geom.Vec2
	rounding radii
	shadow   style.Shadow
	color    color.Color
	outer    bool
	// box and boxRounding are an outer shadow's border box, in its own
	// space, which clips it.
	box         geom.Rect
	boxRounding radii
}

type mesh struct {
	points  []geom.Vec2
	colors  []color.Color
	indices []int
}

// shadowMeshes keeps the meshes made, up to maxMeshes, then starts over.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var (
	shadowMeshes   = map[shadowKey]*mesh{}
	shadowMeshesMu sync.Mutex
)

const maxMeshes = 256

// shadowMesh is the mesh of k's shadow in its box's own space, made once.
func shadowMesh(k shadowKey) *mesh {
	shadowMeshesMu.Lock()
	defer shadowMeshesMu.Unlock()
	if m, ok := shadowMeshes[k]; ok {
		return m
	}
	if len(shadowMeshes) >= maxMeshes {
		clear(shadowMeshes)
	}
	m := buildInsetMesh(k)
	if k.outer {
		m = buildOuterMesh(k)
	}
	shadowMeshes[k] = m
	return m
}

// buildOuterMesh shades each point by the Gaussian blur of the spread
// box: separable for a square one; for a rounded one exact across each
// row, its span's two edges, and summed down the rows within four
// sigmas, an eighth of a sigma apart. Each point is then cleared under
// the border box. Its nodes run three sigmas round the box, every sigma
// (a point and a half at least) through the shadow's corners and four
// sigmas past them, at the border box's edges and through its rounded
// corners every boxStep; nodes closer than minNodeGap merge, and the
// points no triangle uses go.
func buildOuterMesh(k shadowKey) *mesh {
	w, h := k.size.X, k.size.Y
	sigma := k.shadow.Blur / 2
	corner := max(k.rounding[0], k.rounding[1], k.rounding[2], k.rounding[3]) + 4*sigma
	step := max(sigma, 1.5)
	boxCorner := max(k.boxRounding[0], k.boxRounding[1], k.boxRounding[2], k.boxRounding[3])
	nodes := func(length, b0, b1 float32) []float32 {
		lo, hi := -3*sigma, length+3*sigma
		out := blurNodes(lo, hi, 0, length, sigma)
		if !k.rounding.zero() {
			for d := float32(0); d < corner; d += step {
				out = append(out, min(lo+d, hi), max(hi-d, lo))
			}
		}
		for d := boxStep(boxCorner); d < boxCorner; d += boxStep(boxCorner) {
			out = append(out, b0+d, b1-d)
		}
		out = slices.DeleteFunc(out, func(v float32) bool { return v < lo || v > hi })
		slices.Sort(out)
		out = thin(out, minNodeGap)
		out = append(out, b0-clipEdge, b0, b1, b1+clipEdge)
		out = slices.DeleteFunc(out, func(v float32) bool { return v < lo || v > hi })
		slices.Sort(out)
		return slices.Compact(out)
	}
	blur := func(x, y float32) float32 {
		return blurredInterval(x, 0, w, sigma) * blurredInterval(y, 0, h, sigma)
	}
	if !k.rounding.zero() {
		blur = func(x, y float32) float32 {
			lo, hi := max(y-4*sigma, 0), min(y+4*sigma, h)
			if hi <= lo {
				return 0
			}
			n := int(math.Ceil(float64((hi - lo) / (sigma / 8))))
			dt := (hi - lo) / float32(n)
			var sum float64
			for i := range n {
				t := lo + (float32(i)+0.5)*dt
				x0, x1 := span(t, w, h, k.rounding)
				g := float64((t - y) / sigma)
				sum += math.Exp(-g*g/2) * float64(blurredInterval(x, x0, x1, sigma))
			}
			return float32(sum * float64(dt/sigma) / math.Sqrt(2*math.Pi))
		}
	}
	xs, ys := nodes(w, k.box.X, k.box.X+k.box.W), nodes(h, k.box.Y, k.box.Y+k.box.H)
	return gridMesh(xs, ys, k.color, func(x, y float32) float32 {
		clip := min(max(roundedDistance(geom.V2(x, y), k.box, k.boxRounding)/clipEdge, 0), 1)
		if clip == 0 {
			return 0
		}
		return clip * blur(x, y)
	})
}

// minNodeGap is how close two mesh nodes along an axis may be, but for
// the border box's edges, kept exact: closer ones add vertices a linear
// blend doesn't need.
const minNodeGap = 0.75

// boxStep is the gap between nodes through a border box's rounded corner
// of radius r: a point and a half at most, about eight along the arc.
func boxStep(r float32) float32 { return min(max(r/8, 1), 1.5) }

// thin drops nodes closer than gap to the one kept before them, keeping
// the first and the last.
func thin(nodes []float32, gap float32) []float32 {
	if len(nodes) < 3 {
		return nodes
	}
	out := nodes[:1]
	for _, v := range nodes[1 : len(nodes)-1] {
		if v-out[len(out)-1] >= gap {
			out = append(out, v)
		}
	}
	return append(out, nodes[len(nodes)-1])
}

// clipEdge is how far past the border box an outer shadow's clip fades
// in: none to speak of, as Chrome's clip is hard on a box's pixel-aligned
// sides; along a rounded corner the nodes a point apart soften it.
const clipEdge = 1.0 / 1024

// span is where row y of a w×h box rounded by r lies across.
func span(y, w, h float32, r radii) (float32, float32) {
	inset := func(rad, d float32) float32 {
		rad = min(rad, w/2, h/2)
		if d >= rad {
			return 0
		}
		return rad - float32(math.Sqrt(float64(rad*rad-(rad-d)*(rad-d))))
	}
	x0 := max(inset(r[0], y), inset(r[3], h-y))
	x1 := w - max(inset(r[1], y), inset(r[2], h-y))
	return x0, x1
}

func buildInsetMesh(k shadowKey) *mesh {
	pad, rounding, sh, c := geom.Rect{W: k.size.X, H: k.size.Y}, k.rounding, k.shadow, k.color
	hole := grow(geom.Rect{
		X: pad.X + sh.X + sh.Spread, Y: pad.Y + sh.Y + sh.Spread,
		W: pad.W - 2*sh.Spread, H: pad.H - 2*sh.Spread,
	}, 0)
	holeRadii := rounding.grown(-sh.Spread)
	sigma := max(sh.Blur/2, 0.25)
	outer := max(rounding[0], rounding[1], rounding[2], rounding[3])
	inner := max(holeRadii[0], holeRadii[1], holeRadii[2], holeRadii[3]) + 2*sigma
	xs := cornerNodes(blurNodes(pad.X, pad.X+pad.W, hole.X, hole.X+hole.W, sigma),
		pad.X, pad.X+pad.W, outer, hole.X, hole.X+hole.W, inner)
	ys := cornerNodes(blurNodes(pad.Y, pad.Y+pad.H, hole.Y, hole.Y+hole.H, sigma),
		pad.Y, pad.Y+pad.H, outer, hole.Y, hole.Y+hole.H, inner)
	edge := roundedPolygon(pad, rounding)
	shade := func(q geom.Vec2) color.Color {
		k := c
		k.A = c.A * float32(0.5*math.Erfc(-float64(roundedDistance(q, hole, holeRadii)/sigma)/math.Sqrt2))
		return k
	}
	m := &mesh{}
	for j := 0; j+1 < len(ys); j++ {
		for i := 0; i+1 < len(xs); i++ {
			cell := []geom.Vec2{{X: xs[i], Y: ys[j]}, {X: xs[i+1], Y: ys[j]}, {X: xs[i+1], Y: ys[j+1]}, {X: xs[i], Y: ys[j+1]}}
			if !insideAll(cell, pad, rounding) {
				cell = clipConvex(cell, edge)
			}
			if len(cell) < 3 {
				continue
			}
			first := len(m.points)
			for _, q := range cell {
				m.points, m.colors = append(m.points, q), append(m.colors, shade(q))
			}
			for t := 1; t+1 < len(cell); t++ {
				m.indices = append(m.indices, first, first+t, first+t+1)
			}
		}
	}
	return m
}

// inside are the radii of the box edges e inside a box rounded by r: each
// corner less the wider of the two edges it joins, never below 0.
func (r radii) inside(e box.Edges) radii {
	widths := [4]float32{max(e.Left, e.Top), max(e.Right, e.Top), max(e.Right, e.Bottom), max(e.Left, e.Bottom)}
	for i := range r {
		r[i] = max(r[i]-widths[i], 0)
	}
	return r
}

// cornerNodes adds to nodes a point every point of length through the
// corner zones: within outer of lo and hi, and within inner of the hole's
// a and b.
func cornerNodes(nodes []float32, lo, hi, outer, a, b, inner float32) []float32 {
	for _, zone := range [][2]float32{{lo, lo + outer}, {hi - outer, hi}, {a, a + inner}, {b - inner, b}} {
		for v := zone[0]; v <= zone[1]; v++ {
			if v > lo && v < hi {
				nodes = append(nodes, v)
			}
		}
	}
	slices.Sort(nodes)
	return slices.Compact(nodes)
}

// roundedDistance is how far q lies outside r rounded by its radii, less
// than 0 inside.
func roundedDistance(q geom.Vec2, r geom.Rect, rounding radii) float32 {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	k := 0
	switch {
	case q.X >= cx && q.Y < cy:
		k = 1
	case q.X >= cx && q.Y >= cy:
		k = 2
	case q.X < cx && q.Y >= cy:
		k = 3
	}
	rad := min(rounding[k], r.W/2, r.H/2)
	d := geom.V2(geom.Abs(q.X-cx)-(r.W/2-rad), geom.Abs(q.Y-cy)-(r.H/2-rad))
	return geom.V2(max(d.X, 0), max(d.Y, 0)).Len() + min(max(d.X, d.Y), 0) - rad
}

// insideAll reports whether every point of cell lies in r rounded by its
// radii.
func insideAll(cell []geom.Vec2, r geom.Rect, rounding radii) bool {
	for _, q := range cell {
		if roundedDistance(q, r, rounding) > 0 {
			return false
		}
	}
	return true
}

// roundedPolygon is r's outline rounded by its radii, clockwise on screen,
// each arc in segments of about a point and a half.
func roundedPolygon(r geom.Rect, rounding radii) []geom.Vec2 {
	var out []geom.Vec2
	centres := [4]geom.Vec2{
		{X: r.X + rounding[0], Y: r.Y + rounding[0]},
		{X: r.X + r.W - rounding[1], Y: r.Y + rounding[1]},
		{X: r.X + r.W - rounding[2], Y: r.Y + r.H - rounding[2]},
		{X: r.X + rounding[3], Y: r.Y + r.H - rounding[3]},
	}
	for i, from := range [4]float64{math.Pi, 1.5 * math.Pi, 0, 0.5 * math.Pi} {
		rad := rounding[i]
		n := max(int(math.Ceil(float64(rad)/1.5)), 1)
		if rad <= 0 {
			n = 0
		}
		for k := 0; k <= n; k++ {
			a := from + math.Pi/2*float64(k)/float64(max(n, 1))
			out = append(out, geom.V2(centres[i].X+rad*float32(math.Cos(a)), centres[i].Y+rad*float32(math.Sin(a))))
		}
	}
	return out
}

// clipConvex is poly clipped to the convex polygon edge, both clockwise on
// screen.
func clipConvex(poly, edge []geom.Vec2) []geom.Vec2 {
	in := func(p, a, b geom.Vec2) bool { return (b.X-a.X)*(p.Y-a.Y)-(b.Y-a.Y)*(p.X-a.X) >= 0 }
	cross := func(p, q, a, b geom.Vec2) geom.Vec2 {
		d1, d2 := q.Sub(p), b.Sub(a)
		den := d1.X*d2.Y - d1.Y*d2.X
		if den == 0 {
			return p
		}
		t := ((a.X-p.X)*d2.Y - (a.Y-p.Y)*d2.X) / den
		return p.Add(d1.Scale(t))
	}
	out := poly
	for i := range edge {
		a, b := edge[i], edge[(i+1)%len(edge)]
		if a == b || len(out) == 0 {
			continue
		}
		src := out
		out = nil
		for j := range src {
			p, q := src[j], src[(j+1)%len(src)]
			switch pin, qin := in(p, a, b), in(q, a, b); {
			case pin && qin:
				out = append(out, q)
			case pin:
				out = append(out, cross(p, q, a, b))
			case qin:
				out = append(out, cross(p, q, a, b), q)
			}
		}
	}
	return out
}

// grow is r grown by d each side, or shrunk when d is negative, never
// past its centre.
func grow(r geom.Rect, d float32) geom.Rect {
	g := geom.Rect{X: r.X - d, Y: r.Y - d, W: r.W + 2*d, H: r.H + 2*d}
	if g.W < 0 {
		g.X, g.W = r.X+r.W/2, 0
	}
	if g.H < 0 {
		g.Y, g.H = r.Y+r.H/2, 0
	}
	return g
}

// border paints the border: one stroke when every side is alike and the
// box is rounded, else each side square, solid or dashed or dotted; a
// rounded dashed border is drawn solid for now.
func (p *painter) border(r geom.Rect, w box.Edges, radius radii, st *style.Computed, alpha float32) {
	side := func(width float32, s style.BorderStyle, c color.Color) (float32, color.Color) {
		if s == style.BorderNone || width <= 0 {
			return 0, color.Color{}
		}
		return width, fade(c, alpha)
	}
	tw, tc := side(w.Top, st.BorderStyle.Top, st.BorderColor.Top)
	rw, rc := side(w.Right, st.BorderStyle.Right, st.BorderColor.Right)
	bw, bc := side(w.Bottom, st.BorderStyle.Bottom, st.BorderColor.Bottom)
	lw, lc := side(w.Left, st.BorderStyle.Left, st.BorderColor.Left)
	if beveled(st.BorderStyle) {
		p.bevel(r, [4]float32{tw, rw, bw, lw}, [4]color.Color{tc, rc, bc, lc}, st.BorderStyle)
		return
	}
	if tw == rw && rw == bw && bw == lw && tc == rc && rc == bc && bc == lc && !radius.zero() {
		switch {
		case tw <= 0 || tc.A <= 0:
		case radius.uniform():
			p.s.Border(r, radius[0], tw, tc)
		default:
			p.s.BorderCorners(r, radius, tw, tc)
		}
		return
	}
	for _, s := range []struct {
		rect   geom.Rect
		c      color.Color
		style  style.BorderStyle
		across bool
	}{
		{geom.Rect{X: r.X, Y: r.Y, W: r.W, H: tw}, tc, st.BorderStyle.Top, true},
		{geom.Rect{X: r.X + r.W - rw, Y: r.Y, W: rw, H: r.H}, rc, st.BorderStyle.Right, false},
		{geom.Rect{X: r.X, Y: r.Y + r.H - bw, W: r.W, H: bw}, bc, st.BorderStyle.Bottom, true},
		{geom.Rect{X: r.X, Y: r.Y, W: lw, H: r.H}, lc, st.BorderStyle.Left, false},
	} {
		if s.rect.W <= 0 || s.rect.H <= 0 || s.c.A <= 0 {
			continue
		}
		switch s.style {
		case style.BorderDashed, style.BorderDotted:
			p.dashes(s.rect, s.across, s.style == style.BorderDotted, s.c)
		case style.BorderNone, style.BorderSolid, style.BorderInset, style.BorderOutset:
			p.s.Rect(s.rect, s.c)
		}
	}
}

// beveled reports whether a side is inset or outset.
func beveled(s style.Sides[style.BorderStyle]) bool {
	for _, x := range [4]style.BorderStyle{s.Top, s.Right, s.Bottom, s.Left} {
		if x == style.BorderInset || x == style.BorderOutset {
			return true
		}
	}
	return false
}

// trapezoid is a side's two triangles.
//
//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var trapezoid = []int{0, 1, 2, 0, 2, 3}

// bevel paints the sides as Chrome does when one is inset or outset: each
// side a trapezoid mitered into its neighbours, an inset's top and left
// and an outset's bottom and right in their colour's darker shade, its
// other sides in the lighter one; any other style is drawn solid for now.
func (p *painter) bevel(r geom.Rect, w [4]float32, c [4]color.Color, st style.Sides[style.BorderStyle]) {
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	t, rt, b, l := w[0], w[1], w[2], w[3]
	sides := [4][4]geom.Vec2{
		{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1 - rt, Y: y0 + t}, {X: x0 + l, Y: y0 + t}},
		{{X: x1, Y: y0}, {X: x1, Y: y1}, {X: x1 - rt, Y: y1 - b}, {X: x1 - rt, Y: y0 + t}},
		{{X: x1, Y: y1}, {X: x0, Y: y1}, {X: x0 + l, Y: y1 - b}, {X: x1 - rt, Y: y1 - b}},
		{{X: x0, Y: y1}, {X: x0, Y: y0}, {X: x0 + l, Y: y0 + t}, {X: x0 + l, Y: y1 - b}},
	}
	styles := [4]style.BorderStyle{st.Top, st.Right, st.Bottom, st.Left}
	for i := range 4 {
		if w[i] <= 0 || c[i].A <= 0 {
			continue
		}
		col := c[i]
		if styles[i] == style.BorderInset || styles[i] == style.BorderOutset {
			darker, lighter := shades(col)
			col = lighter
			if (styles[i] == style.BorderInset) == (i == 0 || i == 3) {
				col = darker
			}
		}
		p.s.Triangles(sides[i][:], trapezoid, col)
	}
}

// shades are the colours Chromium gives an inset or outset side of colour
// c, its Color::Dark and Color::Light worked in encoded sRGB: scaled by
// (v - 0.33) / v and by min(1, v + 0.33) / v, v its brightest channel;
// black's are #545454 and #a8a8a8.
func shades(c color.Color) (color.Color, color.Color) {
	s := c.SRGB()
	v := max(s[0], s[1], s[2])
	if v <= 0 {
		return color.FromSRGB([3]float32{84.0 / 255, 84.0 / 255, 84.0 / 255}).Alpha(c.A),
			color.FromSRGB([3]float32{168.0 / 255, 168.0 / 255, 168.0 / 255}).Alpha(c.A)
	}
	scaled := func(k float32) color.Color {
		return color.FromSRGB([3]float32{s[0] * k, s[1] * k, s[2] * k}).Alpha(c.A)
	}
	return scaled(max(0, (v-0.33)/v)), scaled(min(1, v+0.33) / v)
}

// dashes draws a dashed or dotted side over r, along X when across, else
// along Y, as Chrome lays them: dashes 2 widths long (3 when 1 wide) with
// gaps near a width (2 when 1 wide), dots a width round with gaps near a
// width, square when 1 wide; as many as fit with one flush at each end,
// the gaps spread evenly. Sides run corner to corner, so corners join.
func (p *painter) dashes(r geom.Rect, across, dotted bool, c color.Color) {
	length, w := r.H, r.W
	if across {
		length, w = r.W, r.H
	}
	dash, gap := 2*w, w
	switch {
	case dotted:
		dash = w
	case w < 2:
		dash, gap = 3*w, 2*w
	}
	n := int(math.Round(float64((length + gap) / (dash + gap))))
	if n < 2 {
		p.s.Rect(r, c)
		return
	}
	step := (length - dash) / float32(n-1)
	for i := range n {
		at := float32(i) * step
		piece := geom.Rect{X: r.X, Y: r.Y + at, W: w, H: dash}
		if across {
			piece = geom.Rect{X: r.X + at, Y: r.Y, W: dash, H: w}
		}
		if dotted && w >= 2 {
			p.s.Circle(piece.Center(), w/2, c)
			continue
		}
		p.s.Rect(piece, c)
	}
}

// outline paints an element's outline outside its border box.
func (p *painter) outline(e entry, alpha float32) {
	b, st := e.b, shown(e.b)
	ring := grow(inkOf(e, st), st.OutlineOffset+st.OutlineWidth)
	if !outlined(b) || p.missed(e, ring) {
		return
	}
	p.cover(b, ring, true)
	c := fade(st.OutlineColor, alpha)
	if c.A <= 0 {
		return
	}
	grow := st.OutlineOffset + st.OutlineWidth
	ps := p.pieces(b, e.at)
	radius := radiiOf(st, b.Size).grown(grow)
	if len(ps) > 1 {
		radius = radii{}
	} else {
		ps[0].rect = geom.Rect{X: e.at.X, Y: e.at.Y, W: b.Size.X, H: b.Size.Y}
	}
	for _, pc := range ps {
		r := p.snap(pc.rect)
		out := geom.Rect{X: r.X - grow, Y: r.Y - grow, W: r.W + 2*grow, H: r.H + 2*grow}
		if radius.uniform() {
			p.s.Border(out, radius[0], st.OutlineWidth, c)
		} else {
			p.s.BorderCorners(out, radius, st.OutlineWidth, c)
		}
	}
}

func outlined(b *box.Box) bool {
	st := shown(b)
	return b.Kind == box.Element && st.Visible && st.OutlineStyle != style.BorderNone && st.OutlineWidth > 0
}

// radii are a box's corner radii, top-left, top-right, bottom-right,
// bottom-left: each corner the smaller of its two radii, all scaled down
// together when two on one side would overlap, as CSS does.
type radii [4]float32

// framed is the size b's background and border frame: its box less a
// table's caption room, which its outline goes round as Chrome's does.
func framed(b *box.Box) geom.Vec2 { return geom.V2(b.Size.X, b.Size.Y-b.Wrap.Vertical()) }

func radiiOf(st *style.Computed, size geom.Vec2) radii {
	corner := func(c style.Size[style.Length]) float32 {
		w, okW := c.Width.Resolve(size.X)
		h, okH := c.Height.Resolve(size.Y)
		if !okW || !okH {
			return 0
		}
		return max(min(w, h), 0)
	}
	rc := st.Radius
	r := radii{corner(rc.TopLeft), corner(rc.TopRight), corner(rc.BottomRight), corner(rc.BottomLeft)}
	f := float32(1)
	sides := [][3]float32{{size.X, r[0], r[1]}, {size.X, r[3], r[2]}, {size.Y, r[0], r[3]}, {size.Y, r[1], r[2]}}
	for _, side := range sides {
		if sum := side[1] + side[2]; sum > side[0] && sum > 0 {
			f = min(f, side[0]/sum)
		}
	}
	for i := range r {
		r[i] *= f
	}
	return r
}

func (r radii) zero() bool    { return r == radii{} }
func (r radii) uniform() bool { return r[0] == r[1] && r[1] == r[2] && r[2] == r[3] }

// grown is r with every rounded corner grown by d, never below 0; square
// corners stay square.
func (r radii) grown(d float32) radii {
	for i := range r {
		if r[i] > 0 {
			r[i] = max(r[i]+d, 0)
		}
	}
	return r
}

// text paints a text run's fragments.
func (p *painter) text(e entry, alpha float32) {
	b, st := e.b, shown(e.b)
	if b.Kind != box.TextRun || !st.Visible || len(b.Lines) == 0 || p.culled(e, textInk(e, st)) {
		return
	}
	c := fade(st.Color, alpha)
	if placeholder(b) {
		c = fade(c, placeholderAlpha)
		if st.HasPlaceholder {
			c = fade(st.PlaceholderColor, alpha)
		}
	}
	if c.A <= 0 {
		return
	}
	face := p.faceOf(st.Font)
	for i := len(st.TextShadows) - 1; i >= 0; i-- {
		sh := st.TextShadows[i]
		sc := sh.Color
		if sh.CurrentColor {
			sc = st.Color
		}
		for _, t := range taps(sh.Blur) {
			tc := fade(fade(sc, alpha), t.weight)
			if tc.A <= 0 {
				continue
			}
			o := p.textOptions(face, st.Font.Size, st.LetterSpacing, tc, st.PixelText)
			for _, f := range b.Lines {
				p.s.Text(geom.V2(e.at.X+f.Rect.X+sh.X+t.at.X, e.at.Y+f.Baseline+sh.Y+t.at.Y), f.Text, o...)
			}
		}
	}
	opts := p.textOptions(face, st.Font.Size, st.LetterSpacing, c, st.PixelText)
	for _, f := range b.Lines {
		p.s.Text(geom.V2(e.at.X+f.Rect.X, e.at.Y+f.Baseline), f.Text, opts...)
	}
}

// textKey is what a text run's drawing options depend on.
type textKey struct {
	face           *twod.Face
	size, tracking float32
	color          color.Color
	pixel          bool
}

// textOpts keeps each textKey's options, made once so a frame boxes none,
// up to maxTextOpts, then starts over.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var (
	textOpts   = map[textKey][]twod.TextOption{}
	textOptsMu sync.Mutex
)

const maxTextOpts = 4096

// textOptions are textOptions', the last ones kept on p, as runs in a
// row mostly share their look.
func (p *painter) textOptions(face *twod.Face, size, tracking float32, c color.Color, pixel bool) []twod.TextOption {
	k := textKey{face, size, tracking, c, pixel}
	if p.lastText.opts == nil || p.lastText.key != k {
		p.lastText.key, p.lastText.opts = k, textOptions(face, size, tracking, c, pixel)
	}
	return p.lastText.opts
}

// textOptions are the options that draw text in face at size, tracked,
// in c, from its baseline, unsmoothed when pixel.
func textOptions(face *twod.Face, size, tracking float32, c color.Color, pixel bool) []twod.TextOption {
	k := textKey{face, size, tracking, c, pixel}
	textOptsMu.Lock()
	defer textOptsMu.Unlock()
	if o, ok := textOpts[k]; ok {
		return o
	}
	if len(textOpts) >= maxTextOpts {
		clear(textOpts)
	}
	o := []twod.TextOption{
		twod.Typeface(face), twod.FontSize(size), twod.Tracking(tracking), twod.Baseline(), twod.Color(c),
	}
	if pixel {
		o = append(o, twod.Pixelated())
	}
	textOpts[k] = o
	return o
}

type tap struct {
	at     geom.Vec2
	weight float32
}

// taps share a text shadow's ink out over copies of the text: one for a
// sharp shadow, else a grid over ±1.5 sigma (half the blur) weighted by
// the Gaussian, so a blur is drawn without blurring glyphs.
func taps(blur float32) []tap {
	if blur <= 0 {
		return sharpTap
	}
	tapsMu.Lock()
	defer tapsMu.Unlock()
	if t, ok := blurTaps[blur]; ok {
		return t
	}
	t := blurredTaps(blur)
	blurTaps[blur] = t
	return t
}

// sharpTap is a sharp shadow's one tap; blurTaps keeps a blurred one's, by
// its blur.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var (
	sharpTap = []tap{{weight: 1}}
	blurTaps = map[float32][]tap{}
	tapsMu   sync.Mutex
)

func blurredTaps(blur float32) []tap {
	sigma := blur / 2
	k := int(min(max(float32(math.Ceil(float64(sigma))), 1), 3))
	step := 1.5 * sigma / float32(k)
	var out []tap
	var sum float32
	for y := -k; y <= k; y++ {
		for x := -k; x <= k; x++ {
			at := geom.V2(float32(x)*step, float32(y)*step)
			w := geom.Exp(-(at.X*at.X + at.Y*at.Y) / (2 * sigma * sigma))
			out = append(out, tap{at, w})
			sum += w
		}
	}
	for i := range out {
		out[i].weight /= sum
	}
	return out
}

// faceOf is the face that draws f: Faces', else Go.
func (p *painter) faceOf(f style.Font) *twod.Face {
	if p.face != nil {
		if face := p.face(f); face != nil {
			return face
		}
	}
	return twod.Regular()
}

// placeholderAlpha is what a placeholder keeps of its colour when no
// ::placeholder rule gives it one.
const placeholderAlpha = 0.54

// placeholder reports whether run is the placeholder its element shows.
func placeholder(run *box.Box) bool {
	if run.Node == nil {
		return false
	}
	for a := run.Parent; a != nil; a = a.Parent {
		if a.Kind == box.Element && a.Node != nil {
			d := dom.Data(a.Node)
			return d.Placeholder && d.Shadow == run.Node
		}
	}
	return false
}

// transform is the element's transform in the root's space: its
// functions in order, about its origin.
func transform(st *style.Computed, at, size geom.Vec2) geom.Transform2D {
	ox, _ := st.TransformOrigin.Width.Resolve(size.X)
	oy, _ := st.TransformOrigin.Height.Resolve(size.Y)
	o := at.Add(geom.V2(ox, oy))
	m := geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: o}
	for _, op := range st.Transform {
		m = m.Mul(transformOp(op, size))
	}
	return m.Mul(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: o.Scale(-1)})
}

func transformOp(op style.TransformOp, size geom.Vec2) geom.Transform2D {
	a := op.A
	switch op.Kind {
	case style.Translate:
		x, _ := op.X.Resolve(size.X)
		y, _ := op.Y.Resolve(size.Y)
		return geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(x, y)}
	case style.ScaleOp:
		return geom.Transform2D{X: geom.V2(a[0], 0), Y: geom.V2(0, a[1])}
	case style.Rotate:
		return geom.Transform2DFrom(geom.Vec2{}, a[0], 0, geom.V2(1, 1))
	case style.Skew:
		return geom.Transform2D{X: geom.V2(1, geom.Tan(a[1])), Y: geom.V2(geom.Tan(a[0]), 1)}
	case style.Matrix:
		return geom.Transform2D{X: geom.V2(a[0], a[1]), Y: geom.V2(a[2], a[3]), Origin: geom.V2(a[4], a[5])}
	}
	return geom.Identity2D()
}

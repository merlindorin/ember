// Package draw is what 2D drawing becomes before the GPU: lists of
// vertices and batches, glyph atlases and offscreen passes. twod fills
// them and gpu draws them; games never see them.
package draw

import (
	"math"
	"slices"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

// Blend is how a batch combines with what is under it, as twod.BlendMode.
type Blend int

const (
	Alpha Blend = iota
	Add
	Multiply
	Screen
	Lighten
	// Occlude keeps, per channel, the larger of what is drawn and what is
	// there, alpha included, unpremultiplied: the occluder picture's
	// groups and heights.
	Occlude
)

// Shape kinds understood by the presenter's fragment shader.
const (
	KindSolid     = 0
	KindText      = 1
	KindRoundRect = 2
	KindOrb       = 3
	KindRing      = 4
	KindBorder    = 5
	KindImage     = 6
	KindGlow      = 7
)

// Vertex is the presenter-facing layout: 16 floats.
type Vertex struct {
	Pos   geom.Vec2
	UV    geom.Vec2
	Color color.Color
	// Shape packs kind, half-size and a shape parameter.
	Shape geom.Vec4
	Extra float32
	// LightUV is where the vertex reads its window's light picture, and
	// LightSpread how: (0, 0) not at all, drawn as is; X −1 at LightUV;
	// else the brightest of the middles of the sides of the box of that
	// half-size round LightUV, all four, or, Y negative, all but the top.
	// Lit names it. The light picture read is the one for its key, packed
	// in LightSpread.X: −1 − k at LightUV, the half-size plus 2k round it,
	// k the occluder groups left out plus 8 times its light class (Key).
	LightUV     geom.Vec2
	LightSpread geom.Vec2
}

// AllGroups is every occluder group: what shades a drawing by default.
const AllGroups = 1<<OccluderGroups - 1

// Shade is the occluder groups whose shadows fall on the vertex: every
// group unless ShadedBy said otherwise.
func (v Vertex) Shade() uint8 { return AllGroups &^ uint8(v.Key()&AllGroups) }

// Class is the vertex's light class: 0, or 1 to MaxLightClasses for one
// under a layer some light Affects (Classes says which).
func (v Vertex) Class() int { return v.Key() >> OccluderGroups }

// Key is what picks the light picture the vertex reads: the groups that
// don't shade it plus 8 times its light class.
func (v Vertex) Key() int {
	switch x := v.LightSpread.X; {
	case x < 0:
		return int(-x-1+0.5) & (MaxKeys - 1)
	case x > 0:
		return int(x/2) & (MaxKeys - 1)
	}
	return 0
}

// MaxLightClasses is how many light classes a window tells apart: past
// it, a drawing takes class 0.
const MaxLightClasses = 15

// MaxKeys is how many keys a vertex can carry: every set of groups left
// out of every light class.
const MaxKeys = (MaxLightClasses + 1) << OccluderGroups

// Keys is a set of keys.
type Keys [MaxKeys / 64]uint64

func (k Keys) Has(key int) bool { return k[key/64]&(1<<(key%64)) != 0 }
func (k *Keys) Add(key int)     { k[key/64] |= 1 << (key % 64) }

// Or is every key of k and o.
func (k Keys) Or(o Keys) Keys {
	for i := range k {
		k[i] |= o[i]
	}
	return k
}

// KeysOf is the set of keys.
func KeysOf(keys ...int) Keys {
	var k Keys
	for _, key := range keys {
		k.Add(key)
	}
	return k
}

// Lit is how the vertex reads the light: 0 not at all, 1 at its LightUV,
// 2 round it (LitAround), 3 round it but its top (LitFacing).
func (v Vertex) Lit() int {
	switch {
	case v.LightSpread.X == 0:
		return 0
	case v.LightSpread.X < 0:
		return 1
	case v.LightSpread.Y < 0:
		return 3
	}
	return 2
}

// Batch is a run of vertices sharing a font atlas, an image and a clip
// rectangle.
type Batch struct {
	Atlas *Font
	// Images are the textures the batch draws, ImageCount of them; an
	// image vertex says which in its Shape.X (see Slot).
	Images     [MaxImages]*render.Texture
	ImageCount int
	First      int
	Count      int
	// Clip, when Clipped, bounds what the batch draws.
	Clip    geom.Rect
	Clipped bool
	// Round, when set, also cuts what the batch draws to a rounded rect
	// (decision 0058): the innermost rounded clip over it.
	Round Rounded
	Blend Blend
	// Mask, when set, is a pass whose alpha the batch shows through.
	Mask *render.Texture
	// alone keeps the batch to itself: a pass's picture or a render
	// texture, which the presenter must draw apart.
	alone bool
}

// MaxImages is how many images one batch draws, each in its own texture
// binding: WebGPU guarantees 16 textures a shader stage, and the font
// atlas and the mask take two.
const MaxImages = 8

// Image is the batch's first image, nil for none.
func (b *Batch) Image() *render.Texture { return b.Images[0] }

// Kind and Slot unpack a vertex's Shape.X: its shape kind, and which of its
// batch's images it shows. They are packed as kind + 16 × slot, so a vertex
// stays the same size.
func Kind(packed float32) int { return int(packed+0.5) & 15 }
func Slot(packed float32) int { return int(packed+0.5) >> 4 }

type List struct {
	Vertices []Vertex
	// Indices are the vertices drawn, three a triangle: a quad's four
	// corners are drawn as two triangles sharing two of them, and the
	// batches' First and Count are ranges of them.
	Indices []uint32
	Batches []Batch
	// SnapVertices rounds every vertex to a whole pixel, for pixel art.
	SnapVertices bool
	time         float32
	clips        []clipState

	transform   geom.Transform2D
	transformed bool
	modulate    color.Color
	modulated   bool
	blend       Blend
	// light is how what is drawn next reads the light picture: lit,
	// toLight the list's points to the picture's 0..1, and anchored, the
	// whole of it at anchor (a LitAt sprite), else at each vertex.
	light    LightMode
	toLight  geom.Transform2D
	anchorUV geom.Vec2
	spreadUV geom.Vec2
	// left is the occluder groups whose shadows don't fall on what is
	// drawn next and class its light class, and Shades holds the key of
	// everything lit drawn: the light pictures the GPU must draw for it.
	left   uint8
	class  uint8
	Shades Keys
	// shared maps a mesh's points to the vertices made for them.
	shared []int
}

// SetTransform places what is drawn next: shapes, text and images are
// given in its space. The identity (or the zero value) draws as given.
func (c *List) SetTransform(t geom.Transform2D) {
	c.transform = t
	c.transformed = t != geom.Identity2D() && t != (geom.Transform2D{})
}

func (c *List) Transform() geom.Transform2D {
	if !c.transformed {
		return geom.Identity2D()
	}
	return c.transform
}

// LightMode is how what a list draws reads its window's light picture:
// not at all, at each point, or at one anchor.
type LightMode int

const (
	Unlit LightMode = iota
	LitPerPixel
	LitAtAnchor
	// LitAround takes the light of the brightest of four points round a
	// box: past the middle of each of its sides.
	LitAround
	// LitFacing is LitAround without the point above the box: a wall's
	// face, which looks down the screen, never lit from behind it.
	LitFacing
)

// SetLight sets how what is drawn next is lit: toLight maps this list's
// points to the light picture's 0..1, and anchor, in this list's points,
// is where a LitAtAnchor drawing takes its light.
func (c *List) SetLight(m LightMode, toLight geom.Transform2D, anchor geom.Vec2) {
	c.light, c.toLight = m, toLight
	if m == LitAtAnchor {
		c.anchorUV = toLight.Xform(anchor)
	}
}

// SetLightAround lights what is drawn next by the brightest of the points
// just past the middles of box's sides, box in this list's points: a tall
// tile round what it blocks light with, lit from whichever side the light
// comes.
func (c *List) SetLightAround(toLight geom.Transform2D, box geom.Rect) {
	c.light, c.toLight = LitAround, toLight
	lo, hi := toLight.Xform(geom.V2(box.X, box.Y)), toLight.Xform(geom.V2(box.X+box.W, box.Y+box.H))
	c.anchorUV = lo.Add(hi).Scale(0.5)
	c.spreadUV = geom.V2(abs(hi.X-lo.X)/2, abs(hi.Y-lo.Y)/2)
}

// SetLightFacing lights what is drawn next by the brightest of the points
// just past box's left, right and bottom sides, box in the space of what
// is drawn: a solid tile's face. It applies only while the list draws
// lit per pixel, and holds until RestoreLight.
func (c *List) SetLightFacing(box geom.Rect) {
	if c.light != LitPerPixel {
		return
	}
	if c.transformed {
		a, b := c.transform.Xform(geom.V2(box.X, box.Y)), c.transform.Xform(geom.V2(box.X+box.W, box.Y+box.H))
		box = geom.Rect{X: min(a.X, b.X), Y: min(a.Y, b.Y), W: abs(b.X - a.X), H: abs(b.Y - a.Y)}
	}
	c.SetLightAround(c.toLight, box)
	c.light = LitFacing
}

// SetShadedBy makes only the shadows of the occluder groups in groups
// fall on what is drawn next, lit: AllGroups by default.
func (c *List) SetShadedBy(groups uint8) { c.left = AllGroups &^ groups }

// SetLightClass makes what is drawn next, lit, read the light picture of
// light class class, 0 to MaxLightClasses: the light of every light but
// those Affecting only layers it isn't under.
func (c *List) SetLightClass(class int) { c.class = uint8(min(max(class, 0), MaxLightClasses)) }

// LightState is how a list lights what it draws, for RestoreLight.
type LightState struct {
	mode             LightMode
	anchor, spreadUV geom.Vec2
	left, class      uint8
}

// Light is how the list lights what it draws now.
func (c *List) Light() LightState {
	return LightState{c.light, c.anchorUV, c.spreadUV, c.left, c.class}
}

// RestoreLight lights what is drawn next as s says, from Light.
func (c *List) RestoreLight(s LightState) {
	c.light, c.anchorUV, c.spreadUV, c.left, c.class = s.mode, s.anchor, s.spreadUV, s.left, s.class
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }

// SetBlend sets how what is drawn next combines with what is under it.
func (c *List) SetBlend(m Blend) { c.blend = m }

// SetModulate multiplies the colors of what is drawn next.
func (c *List) SetModulate(m color.Color) {
	c.modulate = m
	c.modulated = m != color.White && m != (color.Color{})
}

func (c *List) emit(vs ...Vertex) {
	key := int(c.left) + int(c.class)<<OccluderGroups
	k := float32(key)
	if c.light != Unlit {
		c.Shades.Add(key)
	}
	for i := range vs {
		if c.transformed {
			vs[i].Pos = c.transform.Xform(vs[i].Pos)
		}
		if c.SnapVertices {
			vs[i].Pos = geom.V2(float32(math.Round(float64(vs[i].Pos.X))), float32(math.Round(float64(vs[i].Pos.Y))))
		}
		if c.modulated {
			vs[i].Color = vs[i].Color.Mul(c.modulate)
		}
		switch c.light {
		case LitPerPixel:
			vs[i].LightUV, vs[i].LightSpread = c.toLight.Xform(vs[i].Pos), geom.V2(-1-k, 0)
		case LitAtAnchor:
			vs[i].LightUV, vs[i].LightSpread = c.anchorUV, geom.V2(-1-k, 0)
		case LitAround:
			vs[i].LightUV, vs[i].LightSpread = c.anchorUV, geom.V2(
				min(max(c.spreadUV.X, 1e-6), 1)+2*k,
				max(c.spreadUV.Y, 1e-6),
			)
		case LitFacing:
			vs[i].LightUV, vs[i].LightSpread = c.anchorUV, geom.V2(
				min(max(c.spreadUV.X, 1e-6), 1)+2*k,
				-max(c.spreadUV.Y, 1e-6),
			)
		case Unlit:
		}
	}
	c.Vertices = append(c.Vertices, vs...)
}

// vertexIndex is n as a GPU index; a list never holds 2³² vertices.
func vertexIndex(n int) uint32 {
	if n < 0 || n > math.MaxUint32 {
		panic("draw: vertex index out of range")
	}
	return uint32(n)
}

// corners adds the quad of its four corners, top-left, top-right,
// bottom-right then bottom-left, as two triangles.
func (c *List) corners(tl, tr, br, bl Vertex) {
	first := vertexIndex(len(c.Vertices))
	c.emit(tl, tr, br, bl)
	c.Indices = append(c.Indices, first, first+1, first+2, first, first+2, first+3)
}

// Unindexed is the vertices as drawn, three a triangle, in the order of
// the indices: a quad's corners twice where its triangles share them.
func (c *List) Unindexed() []Vertex {
	out := make([]Vertex, len(c.Indices))
	for i, k := range c.Indices {
		out[i] = c.Vertices[k]
	}
	return out
}

// TrianglesColored fills triangles of points, three indices each, each
// point in its own colour, blended across the triangles: a shape whose rim
// fades out, such as a light's reach.
func (c *List) TrianglesColored(points []geom.Vec2, colors []color.Color, indices []int) {
	shape := geom.Vec4{X: KindSolid}
	ok := func(i int) bool { return i >= 0 && i < len(points) && i < len(colors) }
	c.shared = c.shared[:0]
	for range points {
		c.shared = append(c.shared, -1)
	}
	first, n := len(c.Vertices), 0
	at := func(i int) uint32 {
		if c.shared[i] < 0 {
			c.shared[i] = len(c.Vertices)
			c.emit(Vertex{Pos: points[i], Color: colors[i], Shape: shape})
		}
		return vertexIndex(c.shared[i])
	}
	for t := 0; t+3 <= len(indices); t += 3 {
		a, b, d := indices[t], indices[t+1], indices[t+2]
		if !ok(a) || !ok(b) || !ok(d) {
			continue
		}
		c.Indices = append(c.Indices, at(a), at(b), at(d))
		n += 3
	}
	if n > 0 {
		c.batch(nil, len(c.Vertices)-first, n)
	}
}

// Triangles fills triangles of points, three indices each.
func (c *List) Triangles(points []geom.Vec2, indices []int, col color.Color) {
	shape := geom.Vec4{X: KindSolid}
	c.shared = c.shared[:0]
	for range points {
		c.shared = append(c.shared, -1)
	}
	first := len(c.Vertices)
	for _, i := range indices {
		if c.shared[i] < 0 {
			c.shared[i] = len(c.Vertices)
			c.emit(Vertex{Pos: points[i], Color: col, Shape: shape})
		}
		c.Indices = append(c.Indices, vertexIndex(c.shared[i]))
	}
	c.batch(nil, len(c.Vertices)-first, len(indices))
}

func (c *List) Reset(time float32) {
	c.Vertices, c.Indices, c.Batches, c.clips = c.Vertices[:0], c.Indices[:0], c.Batches[:0], c.clips[:0]
	c.time = time
	c.SetTransform(geom.Identity2D())
	c.SetModulate(color.White)
	c.SetBlend(Alpha)
	c.light, c.left, c.class, c.Shades = Unlit, 0, 0, Keys{}
	c.SnapVertices = false
}

// Rounded is a rounded rect a batch is cut to, in the List's space: its
// rect and the radii of its top-left, top-right, bottom-right and
// bottom-left corners. The zero Rounded cuts nothing.
type Rounded struct {
	Rect  geom.Rect
	Radii [4]float32
}

// IsZero reports whether r cuts nothing.
func (r Rounded) IsZero() bool { return r == Rounded{} }

// clipState is one clip of the stack: the rect bounding what is drawn,
// and the innermost rounded rect cutting it.
type clipState struct {
	rect  geom.Rect
	round Rounded
}

// PushClip bounds drawing to r, within the current clip, until PopClip.
// A rounded clip around it still rounds what is drawn.
func (c *List) PushClip(r geom.Rect) {
	if c.transformed {
		r = Bounds(c.transform, r)
	}
	s := clipState{rect: r}
	if n := len(c.clips); n > 0 {
		s.rect, s.round = Intersect(r, c.clips[n-1].rect), c.clips[n-1].round
	}
	c.clips = append(c.clips, s)
}

// PushClipRounded bounds drawing to r with its corners rounded by radii
// (top-left, top-right, bottom-right, bottom-left), within the current
// clip, until PopClip. It replaces a rounded clip around it, which then
// clips as its rect (decision 0058). Under a rotation or a skew it clips
// to r's bounds, square.
func (c *List) PushClipRounded(r geom.Rect, radii [4]float32) {
	if c.transformed && (c.transform.X.Y != 0 || c.transform.Y.X != 0) {
		c.PushClip(r)
		return
	}
	round := Rounded{Rect: r, Radii: radii}
	if c.transformed {
		round.Rect = Bounds(c.transform, r)
		k := min(abs(c.transform.X.X), abs(c.transform.Y.Y))
		for i := range round.Radii {
			round.Radii[i] *= k
		}
		rr := &round.Radii
		if c.transform.X.X < 0 {
			rr[0], rr[1], rr[2], rr[3] = rr[1], rr[0], rr[3], rr[2]
		}
		if c.transform.Y.Y < 0 {
			rr[0], rr[1], rr[2], rr[3] = rr[3], rr[2], rr[1], rr[0]
		}
	}
	if radii == [4]float32{} {
		round = Rounded{}
	}
	c.PushClip(r)
	if !round.IsZero() {
		c.clips[len(c.clips)-1].round = round
	}
}

func (c *List) PopClip() {
	if len(c.clips) > 0 {
		c.clips = c.clips[:len(c.clips)-1]
	}
}

// Bounds is the rectangle around r once moved by t.
func Bounds(t geom.Transform2D, r geom.Rect) geom.Rect {
	lo, hi := geom.V2(math.MaxFloat32, math.MaxFloat32), geom.V2(-math.MaxFloat32, -math.MaxFloat32)
	for _, p := range []geom.Vec2{
		{X: r.X, Y: r.Y},
		{X: r.X + r.W, Y: r.Y},
		{X: r.X, Y: r.Y + r.H},
		{X: r.X + r.W, Y: r.Y + r.H},
	} {
		q := t.Xform(p)
		lo, hi = geom.V2(min(lo.X, q.X), min(lo.Y, q.Y)), geom.V2(max(hi.X, q.X), max(hi.Y, q.Y))
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

// Intersect is the part of a inside b.
func Intersect(a, b geom.Rect) geom.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: max(x1-x0, 0), H: max(y1-y0, 0)}
}

func (c *List) clip() (geom.Rect, bool) {
	if n := len(c.clips); n > 0 {
		return c.clips[n-1].rect, true
	}
	return geom.Rect{}, false
}

// round is the rounded rect the current clip cuts to, zero for none.
func (c *List) round() Rounded {
	if n := len(c.clips); n > 0 {
		return c.clips[n-1].round
	}
	return Rounded{}
}

// Time is the elapsed time, handy for animated widgets.
func (c *List) Time() float32 { return c.time }

func (c *List) batch(atlas *Font, verts, n int) { c.batchImage(atlas, nil, verts, n) }

// batchImage files the last n indices, of the last verts vertices, drawn
// with atlas and img, in the last batch when it can hold them: the same
// blend and clip, room for its font and image. Up to MaxImages images
// share a batch, each vertex marking its image's slot; a render texture,
// which has no pixels of its own, keeps a batch to itself.
func (c *List) batchImage(atlas *Font, img *render.Texture, verts, n int) {
	clip, clipped := c.clip()
	round := c.round()
	alone := img != nil && img.Image == nil
	if len(c.Batches) > 0 && !alone && c.joinLast(atlas, img, verts, n, clip, clipped, round) {
		return
	}
	b := Batch{
		Atlas:   atlas,
		First:   len(c.Indices) - n,
		Count:   n,
		Clip:    clip,
		Clipped: clipped,
		Round:   round,
		Blend:   c.blend,
		alone:   alone,
	}
	if img != nil {
		b.Images[0], b.ImageCount = img, 1
	}
	c.Batches = append(c.Batches, b)
}

// joinLast files the indices in the last batch when it can hold them, and
// reports whether it did.
func (c *List) joinLast(
	atlas *Font,
	img *render.Texture,
	verts, n int,
	clip geom.Rect,
	clipped bool,
	round Rounded,
) bool {
	b := &c.Batches[len(c.Batches)-1]
	fonts := atlas == nil || b.Atlas == atlas || b.Atlas == nil
	slot := b.slot(img)
	if b.alone || !fonts || slot < 0 || b.Blend != c.blend || b.Clipped != clipped || b.Clip != clip ||
		b.Round != round {
		return false
	}
	if atlas != nil {
		b.Atlas = atlas
	}
	if img != nil && slot == b.ImageCount {
		b.Images[slot] = img
		b.ImageCount++
	}
	c.mark(slot, verts)
	b.Count += n
	return true
}

// slot is where img goes in b: its slot if b has it, the next free one,
// 0 for no image, or -1 when b is full.
func (b *Batch) slot(img *render.Texture) int {
	if img == nil {
		return 0
	}
	for i := range b.ImageCount {
		if b.Images[i] == img {
			return i
		}
	}
	if b.ImageCount < MaxImages {
		return b.ImageCount
	}
	return -1
}

// mark puts slot in the last n vertices' Shape.X.
func (c *List) mark(slot, n int) {
	if slot == 0 {
		return
	}
	for i := len(c.Vertices) - n; i < len(c.Vertices); i++ {
		c.Vertices[i].Shape.X += float32(16 * slot)
	}
}

// Image draws the uv part (0..1) of a texture over r, tinted.
func (c *List) Image(r geom.Rect, tex *render.Texture, uv geom.Rect, tint color.Color) {
	c.vertices(r, [4]float32{uv.X, uv.Y, uv.X + uv.W, uv.Y + uv.H}, tint, geom.Vec4{X: KindImage})
	c.batchImage(nil, tex, 4, 6)
}

// ImageCorners draws tex over r with each corner's texture point given:
// top-left, top-right, bottom-right then bottom-left, for flips and turns
// a uv rectangle can't express.
func (c *List) ImageCorners(r geom.Rect, tex *render.Texture, uv [4]geom.Vec2, tint color.Color) {
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	shape := geom.Vec4{X: KindImage}
	c.corners(
		Vertex{Pos: geom.V2(x0, y0), UV: uv[0], Color: tint, Shape: shape},
		Vertex{Pos: geom.V2(x1, y0), UV: uv[1], Color: tint, Shape: shape},
		Vertex{Pos: geom.V2(x1, y1), UV: uv[2], Color: tint, Shape: shape},
		Vertex{Pos: geom.V2(x0, y1), UV: uv[3], Color: tint, Shape: shape},
	)
	c.batchImage(nil, tex, 4, 6)
}

// NinePatch stretches a texture over r but keeps its margins, in texture
// pixels, unscaled: corners stay sharp, edges stretch along, the center
// both ways.
func (c *List) NinePatch(r geom.Rect, tex *render.Texture, left, top, right, bottom float32, tint color.Color) {
	b := tex.Image.Bounds()
	tw, th := float32(b.Dx()), float32(b.Dy())
	xs := [4]float32{r.X, r.X + left, r.X + r.W - right, r.X + r.W}
	ys := [4]float32{r.Y, r.Y + top, r.Y + r.H - bottom, r.Y + r.H}
	us := [4]float32{0, left / tw, 1 - right/tw, 1}
	vs := [4]float32{0, top / th, 1 - bottom/th, 1}
	for j := range 3 {
		for i := range 3 {
			c.Image(geom.Rect{X: xs[i], Y: ys[j], W: xs[i+1] - xs[i], H: ys[j+1] - ys[j]}, tex,
				geom.Rect{X: us[i], Y: vs[j], W: us[i+1] - us[i], H: vs[j+1] - vs[j]}, tint)
		}
	}
}

func (c *List) quad(r geom.Rect, uv [4]float32, col color.Color, shape geom.Vec4, atlas *Font) {
	c.vertices(r, uv, col, shape)
	c.batch(atlas, 4, 6)
}

func (c *List) vertices(r geom.Rect, uv [4]float32, col color.Color, shape geom.Vec4) {
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	u0, v0, u1, v1 := uv[0], uv[1], uv[2], uv[3]
	c.corners(
		Vertex{Pos: geom.V2(x0, y0), UV: geom.V2(u0, v0), Color: col, Shape: shape},
		Vertex{Pos: geom.V2(x1, y0), UV: geom.V2(u1, v0), Color: col, Shape: shape},
		Vertex{Pos: geom.V2(x1, y1), UV: geom.V2(u1, v1), Color: col, Shape: shape},
		Vertex{Pos: geom.V2(x0, y1), UV: geom.V2(u0, v1), Color: col, Shape: shape},
	)
}

// Rect fills a rectangle.
func (c *List) Rect(r geom.Rect, col color.Color) {
	c.quad(r, [4]float32{}, col, geom.Vec4{X: KindSolid}, nil)
}

// RoundRect fills an anti-aliased rounded rectangle.
func (c *List) RoundRect(r geom.Rect, radius float32, col color.Color) {
	c.sdf(r, KindRoundRect, radius, col)
}

// Circle fills an anti-aliased disc.
func (c *List) Circle(center geom.Vec2, radius float32, col color.Color) {
	c.sdf(
		geom.Rect{X: center.X - radius, Y: center.Y - radius, W: 2 * radius, H: 2 * radius},
		KindRoundRect,
		radius,
		col,
	)
}

// Ring strokes a circle outline of the given thickness.
func (c *List) Ring(center geom.Vec2, radius, thickness float32, col color.Color) {
	c.sdf(geom.Rect{X: center.X - radius, Y: center.Y - radius, W: 2 * radius, H: 2 * radius}, KindRing, thickness, col)
}

// Orb draws a glass globe filled with animated liquid up to fill (0..1),
// the classic action-RPG health and mana display.
func (c *List) Orb(center geom.Vec2, radius, fill float32, liquid color.Color) {
	r := geom.Rect{X: center.X - radius, Y: center.Y - radius, W: 2 * radius, H: 2 * radius}
	c.sdf(r, KindOrb, geom.Clamp01(fill), liquid)
}

// Border strokes a rounded rectangle outline.
func (c *List) Border(r geom.Rect, radius, thickness float32, col color.Color) {
	c.sdf(r, KindBorder, radius, col)
	for i := len(c.Vertices) - 4; i < len(c.Vertices); i++ {
		c.Vertices[i].Extra = thickness
	}
}

// Glow fades col out around a rounded rectangle, from its edge to spread
// beyond it, for a soft halo or shadow; it leaves the inside clear.
func (c *List) Glow(r geom.Rect, radius, spread float32, col color.Color) {
	if spread <= 0 {
		return
	}
	hw, hh := r.W/2, r.H/2
	out := geom.Rect{X: r.X - spread, Y: r.Y - spread, W: r.W + 2*spread, H: r.H + 2*spread}
	c.quad(
		out,
		[4]float32{-hw - spread, -hh - spread, hw + spread, hh + spread},
		col,
		geom.Vec4{X: KindGlow, Y: hw, Z: hh, W: radius},
		nil,
	)
	for i := len(c.Vertices) - 4; i < len(c.Vertices); i++ {
		c.Vertices[i].Extra = spread
	}
}

// RoundRectCorners fills a rounded rectangle whose corners have their own
// radii, top-left, top-right, bottom-right then bottom-left.
func (c *List) RoundRectCorners(r geom.Rect, radii [4]float32, col color.Color) {
	c.quadrants(r, KindRoundRect, radii, 0, col)
}

// BorderCorners strokes a rounded rectangle outline whose corners have
// their own radii, in RoundRectCorners's order.
func (c *List) BorderCorners(r geom.Rect, radii [4]float32, thickness float32, col color.Color) {
	c.quadrants(r, KindBorder, radii, thickness, col)
}

// scaleRadii are radii for a w×h box, never negative, all scaled down by
// the same factor when two on one side would overlap, as CSS does.
func scaleRadii(r [4]float32, w, h float32) [4]float32 {
	for i := range r {
		r[i] = max(r[i], 0)
	}
	f := float32(1)
	for _, side := range [][3]float32{{w, r[0], r[1]}, {w, r[3], r[2]}, {h, r[0], r[3]}, {h, r[1], r[2]}} {
		if sum := side[1] + side[2]; sum > side[0] && sum > 0 {
			f = min(f, side[0]/sum)
		}
	}
	if f < 1 {
		for i := range r {
			r[i] *= f
		}
	}
	return r
}

// corners are the four corners as sides, in CSS order: -1 left or top,
// +1 right or bottom.
//
//nolint:gochecknoglobals // a lookup table
var corners = [4][2]float32{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}}

// quadrants draws a rounded shape of kind with a radius per corner, the
// radii scaled as CSS scales them: one quad when they are all alike.
// Otherwise each corner is a square quad at its corner, as big as its
// radius (and the border's thickness), measured against a box twice the
// shape's size centred on its far corner, so the shader never clamps the
// radius; strips measured against the shape's own square box fill the
// rest, or for a border its four sides. Pieces meet where coverage is
// whole, and only the shape's outer edges are padded for antialiasing.
func (c *List) quadrants(r geom.Rect, kind float32, radii [4]float32, extra float32, col color.Color) {
	radii = scaleRadii(radii, r.W, r.H)
	if radii[0] == radii[1] && radii[1] == radii[2] && radii[2] == radii[3] {
		c.sdf(r, kind, radii[0], col)
		if kind == KindBorder {
			for i := len(c.Vertices) - 4; i < len(c.Vertices); i++ {
				c.Vertices[i].Extra = extra
			}
		}
		return
	}
	const pad = piecePad
	w, h := r.W, r.H
	var size [4]float32
	for i := range radii {
		size[i] = radii[i]
		if kind == KindBorder {
			size[i] = max(size[i], min(extra, w/2, h/2))
		}
	}
	c.cornerPieces(r, kind, radii, size, extra, col)
	centre := geom.V2(r.X+w/2, r.Y+h/2)
	square := geom.Vec4{X: kind, Y: w / 2, Z: h / 2}
	if kind == KindBorder {
		t := extra
		edge := func(p geom.Rect) { c.piece(p, centre, square, extra, col) }
		edge(geom.Rect{X: r.X + size[0], Y: r.Y - pad, W: w - size[0] - size[1], H: t + pad})
		edge(geom.Rect{X: r.X + w - t, Y: r.Y + size[1], W: t + pad, H: h - size[1] - size[2]})
		edge(geom.Rect{X: r.X + size[3], Y: r.Y + h - t, W: w - size[3] - size[2], H: t + pad})
		edge(geom.Rect{X: r.X - pad, Y: r.Y + size[0], W: t + pad, H: h - size[0] - size[3]})
		return
	}
	c.rowPieces(r, size, centre, square, extra, col)
}

// piecePad is how far quadrants' pieces reach past the rectangle, so their
// antialiased edges are drawn whole.
const piecePad = 1

// piece draws the part p of a rounded rectangle centred on centre, as shape.
func (c *List) piece(p geom.Rect, centre geom.Vec2, shape geom.Vec4, extra float32, col color.Color) {
	if p.W <= 0 || p.H <= 0 {
		return
	}
	uv := [4]float32{p.X - centre.X, p.Y - centre.Y, p.X + p.W - centre.X, p.Y + p.H - centre.Y}
	c.quad(p, uv, col, shape, nil)
	for j := len(c.Vertices) - 4; j < len(c.Vertices); j++ {
		c.Vertices[j].Extra = extra
	}
}

// cornerPieces draws the corners of r, each size[i] square with its radius.
func (c *List) cornerPieces(r geom.Rect, kind float32, radii, size [4]float32, extra float32, col color.Color) {
	const pad = piecePad
	w, h := r.W, r.H
	for i, q := range corners {
		s := size[i]
		if s <= 0 {
			continue
		}
		x, y := r.X, r.Y
		if q[0] > 0 {
			x = r.X + w - s
		}
		if q[1] > 0 {
			y = r.Y + h - s
		}
		p := geom.Rect{X: x, Y: y, W: s + pad, H: s + pad}
		if q[0] < 0 {
			p.X -= pad
		}
		if q[1] < 0 {
			p.Y -= pad
		}
		far := geom.V2(r.X+w*(1-q[0])/2, r.Y+h*(1-q[1])/2)
		c.piece(p, far, geom.Vec4{X: kind, Y: w, Z: h, W: radii[i]}, extra, col)
	}
}

// rowPieces fills r between its corners with rows, each as wide as the
// corners beside it leave.
func (c *List) rowPieces(
	r geom.Rect,
	size [4]float32,
	centre geom.Vec2,
	square geom.Vec4,
	extra float32,
	col color.Color,
) {
	const pad = piecePad
	w, h := r.W, r.H
	ys := []float32{r.Y, r.Y + size[0], r.Y + size[1], r.Y + h - size[3], r.Y + h - size[2], r.Y + h}
	slices.Sort(ys)
	ys = slices.Compact(ys)
	for k := 0; k+1 < len(ys); k++ {
		y0, y1 := ys[k], ys[k+1]
		mid := (y0 + y1) / 2
		left, right := float32(0), float32(0)
		switch {
		case mid < r.Y+size[0]:
			left = size[0]
		case mid > r.Y+h-size[3]:
			left = size[3]
		}
		switch {
		case mid < r.Y+size[1]:
			right = size[1]
		case mid > r.Y+h-size[2]:
			right = size[2]
		}
		x0, x1 := r.X+left, r.X+w-right
		if left == 0 {
			x0 -= pad
		}
		if right == 0 {
			x1 += pad
		}
		if y0 == r.Y {
			y0 -= pad
		}
		if y1 == r.Y+h {
			y1 += pad
		}
		c.piece(geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, centre, square, extra, col)
	}
}

func (c *List) sdf(r geom.Rect, kind, param float32, col color.Color) {
	const pad = 1
	r = geom.Rect{X: r.X - pad, Y: r.Y - pad, W: r.W + 2*pad, H: r.H + 2*pad}
	hw, hh := r.W/2-pad, r.H/2-pad
	uv := [4]float32{-hw - pad, -hh - pad, hw + pad, hh + pad}
	c.quad(r, uv, col, geom.Vec4{X: kind, Y: hw, Z: hh, W: param}, nil)
}

// Line draws a segment of the given width.
func (c *List) Line(a, b geom.Vec2, width float32, col color.Color) {
	d := b.Sub(a).Normalize()
	n := geom.V2(-d.Y, d.X).Scale(width / 2)
	shape := geom.Vec4{X: KindSolid}
	c.corners(
		Vertex{Pos: a.Add(n), Color: col, Shape: shape},
		Vertex{Pos: b.Add(n), Color: col, Shape: shape},
		Vertex{Pos: b.Sub(n), Color: col, Shape: shape},
		Vertex{Pos: a.Sub(n), Color: col, Shape: shape},
	)
	c.batch(nil, 4, 6)
}

// Gradient fills a rectangle with a vertical gradient.
func (c *List) Gradient(r geom.Rect, top, bottom color.Color) {
	shape := geom.Vec4{X: KindSolid}
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	c.corners(
		Vertex{Pos: geom.V2(x0, y0), Color: top, Shape: shape},
		Vertex{Pos: geom.V2(x1, y0), Color: top, Shape: shape},
		Vertex{Pos: geom.V2(x1, y1), Color: bottom, Shape: shape},
		Vertex{Pos: geom.V2(x0, y1), Color: bottom, Shape: shape},
	)
	c.batch(nil, 4, 6)
}

// Style configures text rendering. Zero values pick sensible defaults.
type Style struct {
	Face   *Face
	Size   float32
	Color  color.Color
	Shadow bool
	Align  Align
	// Tracking is room added after each grapheme cluster, in points.
	Tracking float32
	// Baseline takes the text's origin as its first baseline's left end,
	// not its top-left: text placed by a layout's metrics.
	Baseline bool
	// Pixelated draws the glyphs unsmoothed (Face.PixelSize).
	Pixelated bool
}

type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Font is the face of s at its size.
func (s Style) Font() *Font {
	face, size := s.Face, s.Size
	if face == nil {
		face = Regular
	}
	if size == 0 {
		size = 16
	}
	if s.Pixelated {
		return face.PixelSize(size)
	}
	return face.Size(size)
}

// Measure is the size text takes in s.
func (s Style) Measure(text string) geom.Vec2 { return s.Font().measure(text, s.Tracking) }

// Measure returns the size the text would occupy.
func (c *List) Measure(text string, s Style) geom.Vec2 { return s.Measure(text) }

// Text draws text with its top-left (or top-center/right per Align) at p.
func (c *List) Text(p geom.Vec2, text string, s Style) geom.Vec2 {
	f := s.Font()
	size := f.measure(text, s.Tracking)
	switch s.Align {
	case AlignCenter:
		p.X -= size.X / 2
	case AlignRight:
		p.X -= size.X
	case AlignLeft:
	}
	col := s.Color
	if col == (color.Color{}) {
		col = color.White
	}
	drawn, k := f, float32(1)
	if scale := c.textScale(); scale > 1.01 {
		big := s
		big.Size = f.size * scale
		drawn = big.Font()
		k = drawn.size / f.size
	}
	if s.Shadow {
		c.glyphs(drawn, k, p.Add(geom.V2(1, 1.5)), text, color.RGBA(0, 0, 0, col.A*0.8), false, s)
	}
	c.glyphs(drawn, k, p, text, col, true, s)
	return size
}

// textScale is how much the current transform enlarges what is drawn, so
// text is rasterized that much bigger and stays crisp in a scaled UI.
func (c *List) textScale() float32 {
	t := c.Transform()
	return max(t.X.Len(), t.Y.Len())
}

// glyphs lays text out from p with f, a font k times the size the text is
// drawn at: its metrics are divided by k; s's tracking follows each
// cluster. The baseline lies on a whole atlas pixel and each glyph's quad
// too, the glyph rasterized at the phase of its exact position, so text
// keeps the font's advances and is drawn 1:1, never blurred.
func (c *List) glyphs(f *Font, k float32, p geom.Vec2, text string, col color.Color, pictures bool, s Style) {
	px := oversample * k
	top := p.Y
	if s.Baseline {
		top -= f.ascent / k
	}
	y := float32(math.Round(float64((top+f.ascent/k)*px))) / px
	x := p.X
	shape := geom.Vec4{X: KindText}
	prev := none
	eachCluster(text, func(cl string, first rune) {
		if first == '\n' {
			x = p.X
			y += f.lineHeight / k
			prev = none
			return
		}
		if tex, ok := f.picture(cl); ok {
			if pictures {
				picTop := y - f.ascent/k + (f.lineHeight-f.size)/2/k
				c.Image(
					geom.Rect{X: x, Y: picTop, W: f.size / k, H: f.size / k},
					tex,
					geom.Rect{W: 1, H: 1},
					color.RGBA(1, 1, 1, col.A),
				)
			}
			x += f.size/k + s.Tracking
			prev = none
			return
		}
		g := f.glyph(first)
		x += f.kern(prev, g) / k
		prev = g
		at := float64(x * px)
		whole := math.Floor(at)
		phase := int(math.Round((at - whole) * phases))
		if phase == phases {
			whole, phase = whole+1, 0
		}
		if gp := f.glyphAt(first, phase); gp.size.X > 0 {
			c.quad(
				geom.Rect{
					X: float32(whole)/px + gp.offset.X/k,
					Y: y + gp.offset.Y/k,
					W: gp.size.X / k,
					H: gp.size.Y / k,
				},
				f.uv(gp),
				col,
				shape,
				f,
			)
		}
		x += g.advance/k + s.Tracking
	})
}

// Extent is the rectangle around everything drawn on c, false when
// nothing was.
func (c *List) Extent() (geom.Rect, bool) {
	if len(c.Vertices) == 0 {
		return geom.Rect{}, false
	}
	lo, hi := c.Vertices[0].Pos, c.Vertices[0].Pos
	for _, v := range c.Vertices[1:] {
		lo = geom.V2(min(lo.X, v.Pos.X), min(lo.Y, v.Pos.Y))
		hi = geom.V2(max(hi.X, v.Pos.X), max(hi.Y, v.Pos.Y))
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}, true
}

// Shift moves everything drawn on c, and the batches' clips, by d.
func (c *List) Shift(d geom.Vec2) {
	for i := range c.Vertices {
		c.Vertices[i].Pos = c.Vertices[i].Pos.Add(d)
	}
	for i := range c.Batches {
		if c.Batches[i].Clipped {
			c.Batches[i].Clip.X += d.X
			c.Batches[i].Clip.Y += d.Y
		}
		if !c.Batches[i].Round.IsZero() {
			c.Batches[i].Round.Rect.X += d.X
			c.Batches[i].Round.Rect.Y += d.Y
		}
	}
}

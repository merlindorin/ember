package gpu

import (
	"math"
	"unsafe"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/wgpu"
	"github.com/merlindorin/ember/platform"
)

// staticTexels is a baked tile's side in texels: a texel a point of the
// world, as the light picture is at zoom 2.
const staticTexels = 256

// maxStaticTiles is how many baked tiles are kept; past it, the least
// recently shown give their texture up.
const maxStaticTiles = 128

// staticWGSL draws a picture over a quad: a baked tile copied out of the
// light picture it was drawn in, or a tile added to a window's.
const staticWGSL = `
@group(0) @binding(0) var picture: texture_2d<f32>;
@group(0) @binding(1) var pictureSampler: sampler;

struct Out {
	@builtin(position) pos: vec4f,
	@location(0) uv: vec2f,
};

@vertex
fn static_vs(@location(0) ndc: vec2f, @location(1) uv: vec2f) -> Out {
	return Out(vec4f(ndc, 0.0, 1.0), uv);
}

@fragment
fn static_fs(in: Out) -> @location(0) vec4f {
	return vec4f(textureSampleLevel(picture, pictureSampler, in.uv, 0.0).rgb, 1.0);
}
`

type staticVertex struct{ NDC, UV [2]float32 }

// staticBakes are the baked tiles of static lights, by bake and tile, the
// window they are drawn in as if they were one, and what draws them.
type staticBakes struct {
	tiles    map[staticKey]*bakedTile
	scratch  *platform.Window
	timers   *wgpu.Timers
	copyPipe *wgpu.RenderPipeline
	addPipe  *wgpu.RenderPipeline
	copyBuf  *wgpu.Buffer
	addBuf   *wgpu.Buffer
	verts    []staticVertex
	frame    uint64
	// Baked counts the tiles drawn, for tests and the bench.
	baked int
}

type staticKey struct {
	bake any
	x, y int
}

type bakedTile struct {
	tex     *wgpu.Texture
	group   *wgpu.BindGroup
	version uint64
	shown   uint64
}

func (r *webgpu2D) initStatic() error {
	s := &r.lights.static
	if s.copyPipe != nil {
		return nil
	}
	sh, err := r.dev.CreateShader("static lights", staticWGSL)
	if err != nil {
		return err
	}
	var v staticVertex
	spec := func(label string, blend *wgpu.Blend) wgpu.PipelineSpec {
		return wgpu.PipelineSpec{Label: label, Shader: sh, Layouts: []*wgpu.BindGroupLayout{r.lights.lightSet},
			Vertex: "static_vs", Fragment: "static_fs", Stride: unsafe.Sizeof(v),
			Attributes: []wgpu.Attribute{
				{Location: 0, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(v.NDC)},
				{Location: 1, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(v.UV)},
			},
			Format: lightFormat, Blend: blend}
	}
	if s.copyPipe, err = r.dev.CreateRenderPipeline(spec("static copy", nil)); err != nil {
		return err
	}
	add := wgpu.Blend{
		Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.One},
		Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.One},
	}
	if s.addPipe, err = r.dev.CreateRenderPipeline(spec("static add", &add)); err != nil {
		return err
	}
	s.tiles = map[staticKey]*bakedTile{}
	s.timers = r.dev.NewTimers()
	return nil
}

// bakeStatic draws the tiles of st the GPU doesn't hold at their version,
// each in a submit of its own, before the window's frame.
func (r *webgpu2D) bakeStatic(st *draw.StaticLights) {
	if st == nil || len(st.Tiles) == 0 {
		return
	}
	if err := r.initStatic(); err != nil {
		r.fail(err)
		return
	}
	s := &r.lights.static
	s.frame++
	for _, t := range st.Tiles {
		k := staticKey{st.Key, t.X, t.Y}
		bt := s.tiles[k]
		if bt == nil {
			bt = r.newBakedTile()
			s.tiles[k] = bt
		}
		bt.shown = s.frame
		if bt.tex != nil && bt.version == t.Version && bt.version != 0 {
			continue
		}
		r.bakeTile(bt, t.Bake())
		bt.version = t.Version
		s.baked++
	}
	r.evictStatic()
}

func (r *webgpu2D) newBakedTile() *bakedTile {
	tex := r.dev.CreateTexture(
		staticTexels,
		staticTexels,
		lightFormat,
		wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding,
	)
	group := r.dev.CreateBindGroup(
		r.lights.lightSet,
		wgpu.BindEntry{Binding: 0, View: tex.View()},
		wgpu.BindEntry{Binding: 1, Sampler: r.lights.sampler},
	)
	return &bakedTile{tex: tex, group: group}
}

// bakeTile draws b into bt: its lighting in the scratch window, as a
// window's frame draws it, then its inner rectangle copied into bt.
func (r *webgpu2D) bakeTile(bt *bakedTile, b draw.TileBake) {
	s := &r.lights.static
	l := &r.lights
	w, h := int(math.Ceil(float64(b.Size.X))), int(math.Ceil(float64(b.Size.Y)))
	if s.scratch == nil || s.scratch.Size() != geom.V2(float32(w), float32(h)) {
		if s.scratch != nil {
			if p := l.pictures[s.scratch]; p != nil {
				p.release()
				delete(l.pictures, s.scratch)
			}
		}
		s.scratch = platform.NewWindow(platform.Size(w, h))
	}
	size := s.scratch.Size()
	// A point of the world is a texel of the light picture: twice the
	// window's pixels, since the picture is lightScale of them.
	fb := size.Scale(1 / lightScale)
	e := r.dev.CreateEncoder()
	var occ *span
	if spans := r.plan([]*draw.List{b.Lighting.Occluders}); len(spans) > 0 {
		occ = &spans[0]
	}
	r.light(e, s.scratch, b.Lighting, lightSpans{occluders: occ}, fb, s.timers)
	p := l.pictures[s.scratch]
	rect := draw.LightRect(size)
	u0, v0 := (b.Inner.X-rect.X)/rect.W, (b.Inner.Y-rect.Y)/rect.H
	u1, v1 := (b.Inner.X+b.Inner.W-rect.X)/rect.W, (b.Inner.Y+b.Inner.H-rect.Y)/rect.H
	s.verts = append(
		s.verts[:0],
		quadVerts([4]geom.Vec2{{X: -1, Y: 1}, {X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: -1}}, u0, v0, u1, v1)...)
	bytes := len(s.verts) * int(unsafe.Sizeof(staticVertex{}))
	s.copyBuf = r.grow(s.copyBuf, bytes, wgpu.BufferVertex|wgpu.BufferCopyDst)
	r.dev.WriteBuffer(s.copyBuf, 0, unsafe.Pointer(&s.verts[0]), bytes)
	pass := e.BeginPass(bt.tex.View(), &wgpu.Color{})
	pass.SetPipeline(s.copyPipe)
	pass.SetBindGroup(0, p.plain)
	pass.SetVertexBuffer(s.copyBuf, 0, bytes)
	pass.Draw(len(s.verts), 0)
	pass.End()
	if len(r.slots) > 0 {
		r.dev.WriteBuffer(r.uniforms, 0, unsafe.Pointer(&r.slots[0]), len(r.slots))
	}
	e.Submit()
	l.current = l.unlit
}

// quadVerts are the two triangles of the quad at corners (clockwise from
// the top-left), showing the picture's u0, v0 to u1, v1.
func quadVerts(c [4]geom.Vec2, u0, v0, u1, v1 float32) []staticVertex {
	uv := [4][2]float32{{u0, v0}, {u1, v0}, {u1, v1}, {u0, v1}}
	var out []staticVertex
	for _, i := range [6]int{0, 1, 2, 0, 2, 3} {
		out = append(out, staticVertex{NDC: [2]float32{c[i].X, c[i].Y}, UV: uv[i]})
	}
	return out
}

// evictStatic lets the least recently shown tiles past maxStaticTiles go.
func (r *webgpu2D) evictStatic() {
	s := &r.lights.static
	for len(s.tiles) > maxStaticTiles {
		var oldest staticKey
		var at uint64 = math.MaxUint64
		for k, bt := range s.tiles {
			if bt.shown < at {
				oldest, at = k, bt.shown
			}
		}
		if at == s.frame {
			return
		}
		s.tiles[oldest].release()
		delete(s.tiles, oldest)
	}
}

func (bt *bakedTile) release() {
	bt.group.Release()
	bt.tex.Release()
}

// drawStatic adds the baked tiles of st to the light picture pass draws,
// whose light rectangle is rect, in the window's points.
func (r *webgpu2D) drawStatic(pass *wgpu.Pass, st *draw.StaticLights, rect geom.Rect) {
	s := &r.lights.static
	if st == nil || s.addPipe == nil {
		return
	}
	s.verts = s.verts[:0]
	var groups []*wgpu.BindGroup
	ndc := func(p geom.Vec2) geom.Vec2 {
		return geom.V2((p.X-rect.X)/rect.W*2-1, 1-(p.Y-rect.Y)/rect.H*2)
	}
	for _, t := range st.Tiles {
		bt := s.tiles[staticKey{st.Key, t.X, t.Y}]
		if bt == nil || bt.version == 0 {
			continue
		}
		x, y, n := float32(t.X)*st.TileSize, float32(t.Y)*st.TileSize, st.TileSize
		c := [4]geom.Vec2{
			ndc(st.ToWindow.Xform(geom.V2(x, y))), ndc(st.ToWindow.Xform(geom.V2(x+n, y))),
			ndc(st.ToWindow.Xform(geom.V2(x+n, y+n))), ndc(st.ToWindow.Xform(geom.V2(x, y+n))),
		}
		s.verts = append(s.verts, quadVerts(c, 0, 0, 1, 1)...)
		groups = append(groups, bt.group)
	}
	if len(groups) == 0 {
		return
	}
	bytes := len(s.verts) * int(unsafe.Sizeof(staticVertex{}))
	s.addBuf = r.grow(s.addBuf, bytes, wgpu.BufferVertex|wgpu.BufferCopyDst)
	r.dev.WriteBuffer(s.addBuf, 0, unsafe.Pointer(&s.verts[0]), bytes)
	pass.SetPipeline(s.addPipe)
	pass.SetVertexBuffer(s.addBuf, 0, bytes)
	for i, g := range groups {
		pass.SetBindGroup(0, g)
		pass.Draw(6, 6*i)
	}
}

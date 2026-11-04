package gpu

import (
	"fmt"
	"image"
	imagedraw "image/draw"
	"math"
	"sort"
	"unsafe"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/wgpu"
	"github.com/merlindorin/ember/mesh"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/world"
)

// sceneColor and sceneDepth are the formats the 3D scene draws into: HDR
// colour, and depth.
const (
	sceneColor = wgpu.RGBA16Float
	sceneDepth = wgpu.Depth24Plus
	ldrFormat  = wgpu.RGBA8Unorm
)

// maxLights is how many point lights the scene shader takes.
const maxLights = 32

type batchKey struct {
	mesh *mesh.Data
	mat  *render.Material
}

type lineVertex struct {
	Pos   geom.Vec3
	Color color.Color
}

type billVertex struct {
	Center geom.Vec3
	Corner geom.Vec3
	Color  color.Color
}

// scene3D is the forward HDR renderer: a shadow map, a multisampled scene of lit batches,
// gizmos and billboards over what the 2D layers behind the world drew,
// bloom, the composite, and the camera's post chain.
type scene3D struct {
	settings Settings
	ui       *webgpu2D
	dev      *wgpu.Device
	// plain is the material a mesh without one draws with, and noPost the
	// post a camera without one has.
	plain  *render.Material
	noPost render.Post

	drawables  world.Group[render.Drawable]
	lights     world.Group[render.Light]
	cameras    world.Group[render.Viewer]
	looks      world.Group[render.Lit]
	gizmos     world.Group[render.GizmoSet]
	billboards world.Group[render.BillboardSet]
	neutral    *render.Environment

	shader, post, depthCopy          *wgpu.Shader
	frameSet, shadowSet, materialSet *wgpu.BindGroupLayout
	postSet, copySet, copyOneSet     *wgpu.BindGroupLayout
	pipelines                        map[string]*wgpu.RenderPipeline
	shadowSampler, linear, nearest   *wgpu.Sampler
	materialSamplers                 map[wgpu.SamplerSpec]*wgpu.Sampler
	white                            *wgpu.Texture

	meshes   map[*mesh.Data]*gpuMesh
	textures map[*render.Texture]*gpuMaterialTexture
	batches  map[batchKey]*batch3D
	order    []*batch3D
	surfaces map[*platform.Window]*surface3D
	frame    uint64
	time     float32

	// Per window and frame: instances, gizmo and billboard vertices,
	// material and post uniforms, written once each before the submit.
	instances []gpuInstance
	lineVerts []lineVertex
	billVerts []billVertex
	// billAdd is where the additive billboards start.
	billAdd  int
	uniforms []byte
	buffers  frameBuffers
	release  []func()
	stats    *world.Stats
	counted  uint64
}

type frameBuffers struct {
	instances, lines, bills, uniforms *wgpu.Buffer
}

// gpuInstance is gfx.Instance: a model matrix and a tint.
type gpuInstance struct {
	Model [16]float32
	Tint  [4]float32
}

type batch3D struct {
	key                batchKey
	instances, casters []gpuInstance
	depth              float32
	at, castAt         int
}

type gpuMesh struct {
	vertices, indices *wgpu.Buffer
	count             int
	bounds            geom.AABB
	version           uint64
	lastUsed          uint64
}

type gpuMaterialTexture struct {
	gpu      *wgpu.Texture
	sampler  *wgpu.Sampler
	version  uint64
	lastUsed uint64
}

// surface3D is a window's scene targets.
type surface3D struct {
	shadow, msaa, depth, resolved, depthCopy *wgpu.Texture
	bloom                                    []*wgpu.Texture
	ldr                                      [2]*wgpu.Texture
	w, h, samples                            int
	lastFrame                                uint64
}

func (s *surface3D) releaseTargets() {
	for _, t := range append([]*wgpu.Texture{s.msaa, s.depth, s.resolved, s.depthCopy, s.ldr[0], s.ldr[1]}, s.bloom...) {
		if t != nil {
			t.Release()
		}
	}
	s.msaa, s.depth, s.resolved, s.depthCopy, s.ldr, s.bloom = nil, nil, nil, nil, [2]*wgpu.Texture{}, nil
}

// frameUniforms is sceneWGSL's Frame.
type frameUniforms struct {
	ViewProj, LightVP    [16]float32
	CamPos               [3]float32
	Time                 float32
	SunDir               [3]float32
	Ambient              float32
	SunColor             [3]float32
	FogDensity           float32
	Sky                  [3]float32
	ShadowTexel          float32
	Ground               [3]float32
	Shadows              float32
	FogColor             [3]float32
	LightCount           float32
	Right                [3]float32
	_                    float32
	Up                   [3]float32
	_                    float32
	LightPos, LightColor [maxLights][4]float32
}

// materialUniforms is sceneWGSL's Material.
type materialUniforms struct {
	Albedo                             [4]float32
	Emissive                           [3]float32
	Specular, Wind, Unlit, Textured, _ float32
}

// postUniforms is postWGSL's Post.
type postUniforms struct {
	Texel, Size                                  [2]float32
	Threshold, BloomStrength, Exposure, Vignette float32
	Saturation, Contrast, Time, Nearest          float32
	ShadowTint                                   [3]float32
	_                                            float32
	HighlightTint                                [3]float32
	_                                            float32
}

func newScene3D(s Settings, ui *webgpu2D) *scene3D {
	if s.MSAA == 0 {
		s.MSAA = 4
	}
	if s.ShadowSize == 0 {
		s.ShadowSize = 2048
	}
	if s.RenderScale == 0 {
		s.RenderScale = 1
	}
	return &scene3D{
		settings:         s,
		ui:               ui,
		plain:            render.Mat(color.Gray(0.8)),
		meshes:           map[*mesh.Data]*gpuMesh{},
		textures:         map[*render.Texture]*gpuMaterialTexture{},
		batches:          map[batchKey]*batch3D{},
		surfaces:         map[*platform.Window]*surface3D{},
		pipelines:        map[string]*wgpu.RenderPipeline{},
		materialSamplers: map[wgpu.SamplerSpec]*wgpu.Sampler{},
	}
}

func (r *scene3D) init(d *wgpu.Device) error {
	r.dev = d
	var err error
	if r.shader, err = d.CreateShader("scene", sceneWGSL); err != nil {
		return err
	}
	if r.post, err = d.CreateShader("post", postWGSL); err != nil {
		return err
	}
	if r.depthCopy, err = d.CreateShader("depth copy", depthCopyWGSL); err != nil {
		return err
	}
	both := wgpu.StageVertex | wgpu.StageFragment
	layouts := []struct {
		to      **wgpu.BindGroupLayout
		entries []wgpu.LayoutEntry
	}{
		{
			&r.frameSet,
			[]wgpu.LayoutEntry{
				{Binding: 0, Visibility: both, Kind: wgpu.BindUniform},
				{
					Binding:    1,
					Visibility: wgpu.StageFragment,
					Kind:       wgpu.BindDepth,
				},
				{Binding: 2, Visibility: wgpu.StageFragment, Kind: wgpu.BindSamplerCompare},
			},
		},
		{&r.shadowSet, []wgpu.LayoutEntry{{Binding: 0, Visibility: both, Kind: wgpu.BindUniform}}},
		{
			&r.materialSet,
			[]wgpu.LayoutEntry{
				{Binding: 0, Visibility: both, Kind: wgpu.BindUniform, Dynamic: true},
				{
					Binding:    1,
					Visibility: wgpu.StageFragment,
					Kind:       wgpu.BindTexture,
				},
				{Binding: 2, Visibility: wgpu.StageFragment, Kind: wgpu.BindSampler},
			},
		},
		{
			&r.postSet,
			[]wgpu.LayoutEntry{
				{Binding: 0, Visibility: wgpu.StageFragment, Kind: wgpu.BindUniform},
				{
					Binding:    1,
					Visibility: wgpu.StageFragment,
					Kind:       wgpu.BindTexture,
				},
				{Binding: 2, Visibility: wgpu.StageFragment, Kind: wgpu.BindSampler},
				{
					Binding:    3,
					Visibility: wgpu.StageFragment,
					Kind:       wgpu.BindTexture,
				},
				{Binding: 4, Visibility: wgpu.StageFragment, Kind: wgpu.BindSampler},
			},
		},
		{
			&r.copySet,
			[]wgpu.LayoutEntry{{Binding: 0, Visibility: wgpu.StageFragment, Kind: wgpu.BindDepthMultisampled}},
		},
		{&r.copyOneSet, []wgpu.LayoutEntry{{Binding: 1, Visibility: wgpu.StageFragment, Kind: wgpu.BindDepth}}},
	}
	for _, l := range layouts {
		if *l.to, err = d.CreateBindGroupLayout(l.entries...); err != nil {
			return err
		}
	}
	r.shadowSampler = d.CreateSamplerSpec(wgpu.SamplerSpec{Compare: true})
	r.linear = d.CreateSampler(false, false)
	r.nearest = d.CreateSampler(true, false)
	r.white = d.CreateTexture(1, 1, wgpu.RGBA8UnormSrgb, wgpu.UsageTextureBinding|wgpu.UsageCopyDst)
	d.WriteTexture(r.white, []byte{255, 255, 255, 255}, 4)
	return nil
}

// pipeline is the pipeline called key, made from specOf the first time.
func (r *scene3D) pipeline(key string, specOf func() wgpu.PipelineSpec) *wgpu.RenderPipeline {
	if p, ok := r.pipelines[key]; ok {
		return p
	}
	spec := specOf()
	spec.Label = key
	p, err := r.dev.CreateRenderPipeline(spec)
	if err != nil {
		r.ui.fail(err)
		return nil
	}
	r.pipelines[key] = p
	return p
}

// meshBuffers are a mesh's vertices and its instances' transforms and
// tints, as the lit and shadow pipelines read them.
func meshBuffers() []wgpu.VertexBuffer {
	return []wgpu.VertexBuffer{
		{Stride: unsafe.Sizeof(mesh.Vertex{}), Attributes: []wgpu.Attribute{
			{Location: 0, Format: wgpu.Float32x3, Offset: unsafe.Offsetof(mesh.Vertex{}.Pos)},
			{Location: 1, Format: wgpu.Float32x3, Offset: unsafe.Offsetof(mesh.Vertex{}.Normal)},
			{Location: 2, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(mesh.Vertex{}.UV)},
			{Location: 3, Format: wgpu.Float32x4, Offset: unsafe.Offsetof(mesh.Vertex{}.Color)},
		}},
		{Stride: unsafe.Sizeof(gpuInstance{}), Instance: true, Attributes: []wgpu.Attribute{
			{Location: 4, Format: wgpu.Float32x4, Offset: 0}, {Location: 5, Format: wgpu.Float32x4, Offset: 16},
			{Location: 6, Format: wgpu.Float32x4, Offset: 32}, {Location: 7, Format: wgpu.Float32x4, Offset: 48},
			{Location: 8, Format: wgpu.Float32x4, Offset: unsafe.Offsetof(gpuInstance{}.Tint)},
		}},
	}
}

// litPipeline draws a material's batches: opaque or blended, one or both
// sides.
func (r *scene3D) litPipeline(transparent, doubleSided bool, samples int) *wgpu.RenderPipeline {
	key := fmt.Sprintf("lit t%v d%v s%d", transparent, doubleSided, samples)
	return r.pipeline(key, func() wgpu.PipelineSpec {
		s := wgpu.PipelineSpec{
			Layouts:  []*wgpu.BindGroupLayout{r.frameSet, r.materialSet},
			Shader:   r.shader,
			Vertex:   "lit_vs",
			Fragment: "lit_fs",
			Buffers:  meshBuffers(),
			Format:   sceneColor,
			Samples:  samples,
			Cull:     wgpu.CullBack,
			Depth:    &wgpu.DepthState{Format: sceneDepth, Write: !transparent},
		}
		if doubleSided {
			s.Cull = wgpu.CullNone
		}
		if transparent {
			s.Blend = &wgpu.Blend{
				Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.OneMinusSrcAlpha},
				Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.OneMinusSrcAlpha},
			}
		}
		return s
	})
}

func (r *scene3D) shadowPipeline(doubleSided bool) *wgpu.RenderPipeline {
	return r.pipeline(fmt.Sprintf("shadow d%v", doubleSided), func() wgpu.PipelineSpec {
		s := wgpu.PipelineSpec{
			Layouts: []*wgpu.BindGroupLayout{r.shadowSet, r.materialSet},
			Shader:  r.shader,
			Vertex:  "shadow_vs",
			Buffers: meshBuffers(),
			Cull:    wgpu.CullBack,
			Depth:   &wgpu.DepthState{Format: sceneDepth, Write: true, Bias: 3, SlopeBias: 1.5},
		}
		if doubleSided {
			s.Cull = wgpu.CullNone
		}
		return s
	})
}

// postPipeline is a fullscreen pass running entry into format, adding to
// what is there when additive.
func (r *scene3D) postPipeline(entry string, format wgpu.TextureFormat, additive bool) *wgpu.RenderPipeline {
	return r.pipeline(fmt.Sprintf("post %s %d %v", entry, format, additive), func() wgpu.PipelineSpec {
		s := wgpu.PipelineSpec{
			Layouts:  []*wgpu.BindGroupLayout{r.postSet},
			Shader:   r.post,
			Vertex:   fullVS,
			Fragment: entry,
			Format:   format,
		}
		if additive {
			s.Blend = &wgpu.Blend{
				Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.One},
				Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.One},
			}
		}
		return s
	})
}

// uniform puts v in this frame's uniforms, at an offset a binding may
// take, and gives the offset.
func uniform[T any](r *scene3D, v T) int {
	at := len(r.uniforms)
	size := align(int(unsafe.Sizeof(v)), uniformStride)
	r.uniforms = append(r.uniforms, make([]byte, size)...)
	*(*T)(unsafe.Pointer(&r.uniforms[at])) = v
	return at
}

// group is a bind group for this frame only, let go after its submit.
func (r *scene3D) group(layout *wgpu.BindGroupLayout, entries ...wgpu.BindEntry) *wgpu.BindGroup {
	g := r.dev.CreateBindGroup(layout, entries...)
	r.release = append(r.release, g.Release)
	return g
}

func (r *scene3D) meshFor(d *mesh.Data) *gpuMesh {
	c, ok := r.meshes[d]
	if !ok || c.version != d.Version() {
		if ok {
			c.vertices.Release()
			c.indices.Release()
		}
		c = &gpuMesh{bounds: d.Bounds(), version: d.Version(), count: len(d.Indices)}
		vb, ib := len(d.Vertices)*int(unsafe.Sizeof(mesh.Vertex{})), len(d.Indices)*4
		c.vertices = r.dev.CreateBuffer(align(max(vb, 4), 4), wgpu.BufferVertex|wgpu.BufferCopyDst)
		c.indices = r.dev.CreateBuffer(align(max(ib, 4), 4), wgpu.BufferIndex|wgpu.BufferCopyDst)
		if vb > 0 {
			r.dev.WriteBuffer(c.vertices, 0, unsafe.Pointer(&d.Vertices[0]), vb)
		}
		if ib > 0 {
			r.dev.WriteBuffer(c.indices, 0, unsafe.Pointer(&d.Indices[0]), ib)
		}
		r.meshes[d] = c
	}
	c.lastUsed = r.frame
	return c
}

// textureFor is a material's texture on the GPU, with its mip levels when
// it has Mipmaps, made in linear light from the image.
func (r *scene3D) textureFor(t *render.Texture) *gpuMaterialTexture {
	c, ok := r.textures[t]
	if ok && c.version == t.Version() {
		c.lastUsed = r.frame
		return c
	}
	if ok {
		c.gpu.Release()
	}
	rgba, isRGBA := t.Image.(*image.RGBA)
	if !isRGBA || rgba.Rect.Min != (image.Point{}) {
		b := t.Image.Bounds()
		rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		imagedraw.Draw(rgba, rgba.Bounds(), t.Image, b.Min, imagedraw.Src)
	}
	levels := []*image.RGBA{rgba}
	mips := t.Mipmaps && t.Filter != render.FilterNearest
	if mips {
		for l := rgba; l.Rect.Dx() > 1 || l.Rect.Dy() > 1; {
			l = halve(l)
			levels = append(levels, l)
		}
	}
	g := r.dev.CreateTextureSpec(
		wgpu.TextureSpec{Width: rgba.Rect.Dx(), Height: rgba.Rect.Dy(), Format: wgpu.RGBA8UnormSrgb,
			Usage: wgpu.UsageTextureBinding | wgpu.UsageCopyDst, Mips: len(levels)},
	)
	for i, l := range levels {
		r.dev.WriteTextureLevel(g, i, l.Rect.Dx(), l.Rect.Dy(), l.Pix, l.Stride)
	}
	spec := wgpu.SamplerSpec{Nearest: t.Filter == render.FilterNearest, Repeat: t.Repeat, Mipmaps: mips}
	smp, ok := r.materialSamplers[spec]
	if !ok {
		smp = r.dev.CreateSamplerSpec(spec)
		r.materialSamplers[spec] = smp
	}
	c = &gpuMaterialTexture{gpu: g, sampler: smp, version: t.Version(), lastUsed: r.frame}
	r.textures[t] = c
	return c
}

// halve is img at half its size, each texel the mean of four in linear
// light, as sRGB mipmaps are made.
func halve(img *image.RGBA) *image.RGBA {
	w, h := max(img.Rect.Dx()/2, 1), max(img.Rect.Dy()/2, 1)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			var sum [4]float64
			n := 0.0
			for dy := range 2 {
				for dx := range 2 {
					sx, sy := min(2*x+dx, img.Rect.Dx()-1), min(2*y+dy, img.Rect.Dy()-1)
					p := img.Pix[sy*img.Stride+sx*4:]
					for c := range 3 {
						sum[c] += float64(color.SRGBToLinear(float32(p[c]) / 255))
					}
					sum[3] += float64(p[3]) / 255
					n++
				}
			}
			q := out.Pix[y*out.Stride+x*4:]
			for c := range 3 {
				q[c] = uint8(math.Round(float64(color.LinearToSRGB(float32(sum[c]/n))) * 255))
			}
			q[3] = uint8(math.Round(sum[3] / n * 255))
		}
	}
	return out
}

func (r *scene3D) evict() {
	if r.frame%120 != 0 {
		return
	}
	for d, c := range r.meshes {
		if r.frame-c.lastUsed > 600 {
			c.vertices.Release()
			c.indices.Release()
			delete(r.meshes, d)
		}
	}
	for t, c := range r.textures {
		if r.frame-c.lastUsed > 600 {
			c.gpu.Release()
			delete(r.textures, t)
		}
	}
	for w, s := range r.surfaces {
		if r.frame-s.lastFrame > 120 {
			s.releaseTargets()
			if s.shadow != nil {
				s.shadow.Release()
			}
			delete(r.surfaces, w)
		}
	}
}

// surface is win's targets for a scene of w by h pixels at samples.
func (r *scene3D) surface(win *platform.Window, w, h, samples int) *surface3D {
	s := r.surfaces[win]
	if s == nil {
		s = &surface3D{
			shadow: r.dev.CreateTexture(
				r.settings.ShadowSize,
				r.settings.ShadowSize,
				sceneDepth,
				wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding,
			),
		}
		r.surfaces[win] = s
	}
	s.lastFrame = r.frame
	if s.w == w && s.h == h && s.samples == samples && s.resolved != nil {
		return s
	}
	s.releaseTargets()
	target := wgpu.UsageRenderAttachment | wgpu.UsageTextureBinding
	s.resolved = r.dev.CreateTexture(w, h, sceneColor, target)
	s.depth = r.dev.CreateTextureSpec(
		wgpu.TextureSpec{Width: w, Height: h, Format: sceneDepth, Usage: target, Samples: max(samples, 1)},
	)
	if samples > 1 {
		s.msaa = r.dev.CreateTextureSpec(
			wgpu.TextureSpec{
				Width:   w,
				Height:  h,
				Format:  sceneColor,
				Usage:   wgpu.UsageRenderAttachment,
				Samples: samples,
			},
		)
	}
	for i := range s.ldr {
		s.ldr[i] = r.dev.CreateTexture(w, h, ldrFormat, target)
	}
	bw, bh := w/2, h/2
	for i := 0; i < 6 && bw > 8 && bh > 8; i++ {
		s.bloom = append(s.bloom, r.dev.CreateTexture(bw, bh, sceneColor, target))
		bw, bh = bw/2, bh/2
	}
	s.w, s.h, s.samples = w, h, samples
	return s
}

// environment is the first environment entity, else a neutral one.
func (r *scene3D) environment() *render.Environment {
	for e := range r.looks.All() {
		return e.GetEnvironment()
	}
	if r.neutral == nil {
		r.neutral = render.NewEnvironment()
	}
	return r.neutral
}

// presented is the view and post of the camera win presents, an inactive
// view when it has none.
func (r *scene3D) presented(win *platform.Window) (*render.View, *render.Post) {
	for c := range r.cameras.All() {
		cam := c.GetCamera()
		if cam.Presenting() && cam.RendersIn() == win {
			if cam.Post != nil {
				return cam.View(), cam.Post
			}
			return cam.View(), &r.noPost
		}
	}
	return &render.View{}, &r.noPost
}

// active is whether win shows a 3D view this frame.
func (r *scene3D) active(win *platform.Window) bool {
	v, _ := r.presented(win)
	return v.Active
}

func (r *scene3D) shadowMatrix(env *render.Environment, view *render.View) geom.Mat4 {
	d := env.Shadows()
	if d == 0 {
		d = 30
	}
	texel := 2 * d / float32(r.settings.ShadowSize)
	center := view.Focus
	center.X = geom.Floor(center.X/texel) * texel
	center.Z = geom.Floor(center.Z/texel) * texel
	sunDir, _, _ := env.Sun()
	sun := sunDir.Normalize()
	up := geom.Up
	if geom.Abs(sun.Y) > 0.99 {
		up = geom.Forward
	}
	lv := geom.Mat4LookAt(center.Sub(sun.Scale(150)), center, up)
	return geom.Ortho(-d, d, -d, d, 1, 300).Mul(lv)
}

// collect files the visible drawables and the shadow casters by mesh and
// material, as Renderer.collect does, and lays their instances out in
// this frame's instance buffer.
func (r *scene3D) collect(view *render.View, lightVP geom.Mat4) {
	for _, b := range r.batches {
		b.instances, b.casters = b.instances[:0], b.casters[:0]
	}
	r.order = r.order[:0]
	frustum := view.Frustum()
	lightFrustum := geom.FrustumFromMatrix(lightVP)
	for _, d := range r.drawables.Slice() {
		m := d.GetModel()
		if m.Hidden || m.Mesh == nil || len(m.Mesh.Indices) == 0 {
			continue
		}
		mat := m.Material
		if mat == nil {
			mat = r.plain
		}
		w := d.GetTransform().World()
		cm := r.meshFor(m.Mesh)
		bounds := cm.bounds.Transform(w)
		visible := frustum.IntersectsAABB(bounds)
		casts := !m.NoShadow && !mat.Unlit && !mat.Transparent && lightFrustum.IntersectsAABB(bounds)
		if !visible && !casts {
			continue
		}
		key := batchKey{m.Mesh, mat}
		b, ok := r.batches[key]
		if !ok {
			b = &batch3D{key: key}
			r.batches[key] = b
		}
		if len(b.instances) == 0 && len(b.casters) == 0 {
			r.order = append(r.order, b)
			b.depth = 0
		}
		tint := m.Tint
		if tint == (color.Color{}) {
			tint = color.White
		}
		inst := gpuInstance{Model: w, Tint: tint.Array()}
		if visible {
			b.instances = append(b.instances, inst)
			b.depth = max(b.depth, bounds.Center().Dist(view.Position))
		}
		if casts {
			b.casters = append(b.casters, inst)
		}
	}
	for key, b := range r.batches {
		if len(b.instances) == 0 && len(b.casters) == 0 && r.meshes[key.mesh] == nil {
			delete(r.batches, key)
		}
	}
	r.instances = r.instances[:0]
	for _, b := range r.order {
		b.at = len(r.instances)
		r.instances = append(r.instances, b.instances...)
		b.castAt = len(r.instances)
		r.instances = append(r.instances, b.casters...)
	}
}

// lights are the point lights nearest the focus that the view sees, at
// most maxLights, as Renderer.uploadLights chooses them.
func (r *scene3D) lightsFor(view *render.View, f *frameUniforms) {
	type cand struct {
		pos, color geom.Vec4
		dist       float32
	}
	frustum := view.Frustum()
	var cands []cand
	for _, l := range r.lights.Slice() {
		pl := l.GetPointLight()
		pos := l.GetTransform().WorldPosition()
		rng := pl.Range
		if rng == 0 {
			rng = 8
		}
		if pl.Intensity <= 0 || !frustum.IntersectsSphere(pos, rng) {
			continue
		}
		intensity := pl.Intensity
		if pl.Flicker > 0 {
			t := r.time*9 + pos.X*1.3 + pos.Z*0.7
			intensity *= 1 - pl.Flicker*(0.5+0.25*geom.Sin(t)+0.25*geom.Sin(t*2.3+1.7))
		}
		c := pl.Color.Scale(intensity)
		cands = append(cands, cand{pos.Vec4(rng), geom.V4(c.R, c.G, c.B, 1), pos.Dist(view.Focus)})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].dist < cands[j].dist })
	if len(cands) > maxLights {
		cands = cands[:maxLights]
	}
	f.LightCount = float32(len(cands))
	for i, c := range cands {
		f.LightPos[i] = [4]float32{c.pos.X, c.pos.Y, c.pos.Z, c.pos.W}
		f.LightColor[i] = [4]float32{c.color.X, c.color.Y, c.color.Z, c.color.W}
	}
}

func v3(v geom.Vec3) [3]float32    { return [3]float32{v.X, v.Y, v.Z} }
func rgb(c color.Color) [3]float32 { return [3]float32{c.R, c.G, c.B} }

// draw records win's 3D frame into e, ending on target: the shadows, the
// scene over back (the 2D layers behind the world), bloom, the composite
// and the post chain. It reports whether there was a view to draw.
func (r *scene3D) draw(
	w *world.World,
	e *wgpu.Encoder,
	win *platform.Window,
	target *wgpu.Texture,
	back *span,
	timers *wgpu.Timers,
) bool {
	view, post := r.presented(win)
	if !view.Active {
		return false
	}
	t := w.Time()
	if t.Frame != r.frame {
		r.frame = t.Frame
		r.evict()
	}
	r.time = t.Elapsed
	r.stats = w.Stats()
	if r.counted != r.frame {
		r.counted = r.frame
		r.stats.DrawCalls, r.stats.Triangles = 0, 0
	}
	env := r.environment()
	samples := r.settings.MSAA
	if post.MSAA > 0 {
		samples = post.MSAA
	}
	samples = map[bool]int{true: 4, false: 1}[samples > 1]
	fb := geom.V2(float32(target.Width), float32(target.Height))
	sw, sh := post.SceneSize(fb, r.settings.RenderScale)
	s := r.surface(win, sw, sh, samples)
	lightVP := r.shadowMatrix(env, view)
	r.collect(view, lightVP)
	r.uniforms = r.uniforms[:0]

	frame := r.frameUniforms(env, view, lightVP)
	frameAt := uniform(r, frame)

	materialAt := r.materialSlots()
	r.gizmoVertices()
	r.billboardVertices()

	// Everything this frame writes is laid out above; now the buffers can
	// be grown before anything refers to them.
	r.growBuffers()
	posts := r.postUniforms(env, s, post, view, fb)
	r.writeBuffers()

	frameEntry := wgpu.BindEntry{
		Binding: 0,
		Buffer:  r.buffers.uniforms,
		Offset:  frameAt,
		Size:    int(unsafe.Sizeof(frameUniforms{})),
	}
	dr := r.drawer(materialAt)

	if frame.Shadows > 0 {
		r.shadowPass(e, s, frameEntry, dr, timers)
	}

	// The scene, over what the 2D layers behind the world drew.
	p := beginScene(e, s, win, env, timers)
	if back != nil {
		r.ui.record(
			p,
			*back,
			win.Size(),
			geom.V2(float32(sw), float32(sh)),
			target2D{format: sceneColor, samples: samples, depth: true, exposure: max(env.Exposure(), 1e-3)},
			nil,
		)
	}
	p.SetBindGroup(
		0,
		r.group(
			r.frameSet,
			frameEntry,
			wgpu.BindEntry{Binding: 1, View: s.shadow.View()},
			wgpu.BindEntry{Binding: 2, Sampler: r.shadowSampler},
		),
	)
	tris := r.sceneBatches(p, dr, samples)
	drawCalls := dr.calls + r.drawGizmos(p, samples) + r.drawBillboards(p, samples)
	p.End()

	if !post.Plain() {
		r.copyDepth(e, s)
	}
	drawCalls += r.bloomPasses(e, s, posts)
	drawCalls += r.finish(e, s, post, view, target, posts)
	r.stats.DrawCalls += drawCalls
	r.stats.Triangles += tris
	return true
}

// beginScene begins the scene's pass on s, cleared to win's background,
// resolving s's samples when it has them.
func beginScene(
	e *wgpu.Encoder,
	s *surface3D,
	win *platform.Window,
	env *render.Environment,
	timers *wgpu.Timers,
) *wgpu.Pass {
	bg := hdrBackground(win.Background(), env.Exposure())
	color, resolve := s.resolved.View(), wgpu.TextureView{}
	if s.msaa != nil {
		color, resolve = s.msaa.View(), s.resolved.View()
	}
	p := e.BeginPassSpec(
		wgpu.PassSpec{
			Color:      color,
			Resolve:    resolve,
			Clear:      &wgpu.Color{R: float64(bg[0]), G: float64(bg[1]), B: float64(bg[2]), A: 1},
			Depth:      s.depth.View(),
			DepthClear: 1,
			Span:       timers.Span("scene"),
		},
	)
	return p
}

// growBuffers grows the instance, line and billboard buffers to what this
// frame laid out.
func (r *scene3D) growBuffers() {
	r.buffers.instances = r.ui.grow(
		r.buffers.instances,
		len(r.instances)*int(unsafe.Sizeof(gpuInstance{})),
		wgpu.BufferVertex|wgpu.BufferCopyDst,
	)
	r.buffers.lines = r.ui.grow(
		r.buffers.lines,
		len(r.lineVerts)*int(unsafe.Sizeof(lineVertex{})),
		wgpu.BufferVertex|wgpu.BufferCopyDst,
	)
	r.buffers.bills = r.ui.grow(
		r.buffers.bills,
		len(r.billVerts)*int(unsafe.Sizeof(billVertex{})),
		wgpu.BufferVertex|wgpu.BufferCopyDst,
	)
}

// writeBuffers grows the uniform buffer to this frame's uniforms and
// writes them and the instances, lines and billboards.
func (r *scene3D) writeBuffers() {
	r.buffers.uniforms = r.ui.grow(r.buffers.uniforms, len(r.uniforms), wgpu.BufferUniform|wgpu.BufferCopyDst)
	if len(r.instances) > 0 {
		r.dev.WriteBuffer(
			r.buffers.instances,
			0,
			unsafe.Pointer(&r.instances[0]),
			len(r.instances)*int(unsafe.Sizeof(gpuInstance{})),
		)
	}
	if len(r.lineVerts) > 0 {
		r.dev.WriteBuffer(
			r.buffers.lines,
			0,
			unsafe.Pointer(&r.lineVerts[0]),
			len(r.lineVerts)*int(unsafe.Sizeof(lineVertex{})),
		)
	}
	if len(r.billVerts) > 0 {
		r.dev.WriteBuffer(
			r.buffers.bills,
			0,
			unsafe.Pointer(&r.billVerts[0]),
			len(r.billVerts)*int(unsafe.Sizeof(billVertex{})),
		)
	}
	r.dev.WriteBuffer(r.buffers.uniforms, 0, unsafe.Pointer(&r.uniforms[0]), len(r.uniforms))
}

// drawer3D draws the frame's batches, each with its material's bind group,
// counting its draw calls.
type drawer3D struct {
	r          *scene3D
	groups     map[*render.Material]*wgpu.BindGroup
	materialAt map[*render.Material]int
	calls      int
}

// drawer makes each material's bind group, its uniforms at materialAt.
func (r *scene3D) drawer(materialAt map[*render.Material]int) *drawer3D {
	dr := &drawer3D{r: r, groups: map[*render.Material]*wgpu.BindGroup{}, materialAt: materialAt}
	for m := range materialAt {
		view, smp := r.white.View(), r.linear
		if m.Texture != nil && m.Texture.Image != nil {
			c := r.textureFor(m.Texture)
			view, smp = c.gpu.View(), c.sampler
		}
		dr.groups[m] = r.group(
			r.materialSet,
			wgpu.BindEntry{Binding: 0, Buffer: r.buffers.uniforms, Size: int(unsafe.Sizeof(materialUniforms{}))},
			wgpu.BindEntry{Binding: 1, View: view},
			wgpu.BindEntry{Binding: 2, Sampler: smp},
		)
	}
	return dr
}

// batch draws n instances of b from the at'th.
func (dr *drawer3D) batch(p *wgpu.Pass, b *batch3D, at, n int) {
	r := dr.r
	cm := r.meshes[b.key.mesh]
	p.SetBindGroup(
		1,
		dr.groups[b.key.mat],
		uint32(dr.materialAt[b.key.mat]), //nolint:gosec // a slot's offset, within the uniform buffer
	)
	p.SetVertexBufferSlot(0, cm.vertices, 0, cm.vertices.Size)
	stride := int(unsafe.Sizeof(gpuInstance{}))
	p.SetVertexBufferSlot(1, r.buffers.instances, at*stride, n*stride)
	p.SetIndexBuffer(cm.indices, 0, cm.count*4)
	p.DrawIndexedInstanced(cm.count, 0, n)
	dr.calls++
}

// shadowPass draws the batches that cast into the sun's shadow map.
func (r *scene3D) shadowPass(
	e *wgpu.Encoder,
	s *surface3D,
	frameEntry wgpu.BindEntry,
	dr *drawer3D,
	timers *wgpu.Timers,
) {
	p := e.BeginPassSpec(wgpu.PassSpec{Depth: s.shadow.View(), DepthClear: 1, Span: timers.Span("shadows")})
	p.SetBindGroup(0, r.group(r.shadowSet, frameEntry))
	for _, b := range r.order {
		if len(b.casters) == 0 {
			continue
		}
		if pl := r.shadowPipeline(b.key.mat.DoubleSided); pl != nil {
			p.SetPipeline(pl)
			dr.batch(p, b, b.castAt, len(b.casters))
		}
	}
	p.End()
}

// sceneBatches draws the opaque batches, then the transparent ones back to
// front, and gives the triangles the opaque ones drew.
func (r *scene3D) sceneBatches(p *wgpu.Pass, dr *drawer3D, samples int) int {
	tris := 0
	var transparent []*batch3D
	for _, b := range r.order {
		if len(b.instances) == 0 {
			continue
		}
		if b.key.mat.Transparent {
			transparent = append(transparent, b)
			continue
		}
		if pl := r.litPipeline(false, b.key.mat.DoubleSided, samples); pl != nil {
			p.SetPipeline(pl)
			dr.batch(p, b, b.at, len(b.instances))
			tris += r.meshes[b.key.mesh].count / 3 * len(b.instances)
		}
	}
	sort.Slice(transparent, func(i, j int) bool { return transparent[i].depth > transparent[j].depth })
	for _, b := range transparent {
		if pl := r.litPipeline(true, b.key.mat.DoubleSided, samples); pl != nil {
			p.SetPipeline(pl)
			dr.batch(p, b, b.at, len(b.instances))
		}
	}
	return tris
}

// frameUniforms are the frame's camera, sun, sky, fog and lights, as the
// shaders read them.
func (r *scene3D) frameUniforms(env *render.Environment, view *render.View, lightVP geom.Mat4) frameUniforms {
	frame := frameUniforms{ViewProj: view.ViewProj, LightVP: lightVP, CamPos: v3(view.Position), Time: r.time,
		ShadowTexel: 1 / float32(r.settings.ShadowSize)}
	sunDir, sunColor, sunIntensity := env.Sun()
	sky, ground, ambient := env.Ambient()
	fogColor, fogDensity := env.Fog()
	frame.SunDir, frame.SunColor = v3(sunDir.Normalize()), rgb(sunColor.Scale(sunIntensity))
	frame.Sky, frame.Ground, frame.Ambient = rgb(sky), rgb(ground), ambient
	frame.FogColor, frame.FogDensity = rgb(fogColor), fogDensity
	if env.Shadows() > 0 {
		frame.Shadows = 1
	}
	right, up := view.ViewMat.Transpose().MulDir(geom.Right), view.ViewMat.Transpose().MulDir(geom.Up)
	frame.Right, frame.Up = v3(right), v3(up)
	r.lightsFor(view, &frame)
	return frame
}

// materialSlots writes each drawn material's uniforms once, and gives
// where each lies.
func (r *scene3D) materialSlots() map[*render.Material]int {
	materialAt := map[*render.Material]int{}
	for _, b := range r.order {
		m := b.key.mat
		if _, ok := materialAt[m]; ok {
			continue
		}
		u := materialUniforms{Albedo: m.Albedo.Array(), Emissive: rgb(m.Emissive), Specular: m.Specular, Wind: m.Wind}
		if m.Unlit {
			u.Unlit = 1
		}
		if m.Texture != nil && m.Texture.Image != nil {
			u.Textured = 1
		}
		materialAt[m] = uniform(r, u)
	}
	return materialAt
}

// gizmoVertices are this frame's gizmo lines.
func (r *scene3D) gizmoVertices() {
	r.lineVerts = r.lineVerts[:0]
	for e := range r.gizmos.All() {
		if g := e.GetGizmos(); g.Enabled {
			for _, l := range g.Lines {
				r.lineVerts = append(r.lineVerts, lineVertex{l.A, l.Color}, lineVertex{l.B, l.Color})
			}
		}
	}
}

// billboardVertices are this frame's billboards, the blended ones first,
// then the additive ones from r.billAdd on.
func (r *scene3D) billboardVertices() {
	r.billVerts = r.billVerts[:0]
	r.billAdd = 0
	for _, additive := range []bool{false, true} {
		if additive {
			r.billAdd = len(r.billVerts)
		}
		for e := range r.billboards.All() {
			for _, b := range e.GetBillboards().Items {
				if b.Additive != additive {
					continue
				}
				h := b.Size / 2
				for _, c := range [6][2]float32{{-h, -h}, {h, -h}, {h, h}, {-h, -h}, {h, h}, {-h, h}} {
					r.billVerts = append(r.billVerts, billVertex{b.Position, geom.V3(c[0], c[1], b.Rotation), b.Color})
				}
			}
		}
	}
}

func (r *scene3D) drawGizmos(p *wgpu.Pass, samples int) int {
	if len(r.lineVerts) == 0 {
		return 0
	}
	pl := r.pipeline(fmt.Sprintf("lines s%d", samples), func() wgpu.PipelineSpec {
		return wgpu.PipelineSpec{
			Layouts:  []*wgpu.BindGroupLayout{r.frameSet},
			Shader:   r.shader,
			Vertex:   "line_vs",
			Fragment: "line_fs",
			Buffers: []wgpu.VertexBuffer{{Stride: unsafe.Sizeof(lineVertex{}), Attributes: []wgpu.Attribute{
				{Location: 0, Format: wgpu.Float32x3, Offset: unsafe.Offsetof(lineVertex{}.Pos)},
				{Location: 1, Format: wgpu.Float32x4, Offset: unsafe.Offsetof(lineVertex{}.Color)}}}},
			Format:   sceneColor,
			Samples:  samples,
			Topology: wgpu.Lines,
			Depth:    &wgpu.DepthState{Format: sceneDepth, Write: true},
		}
	})
	if pl == nil {
		return 0
	}
	p.SetPipeline(pl)
	p.SetVertexBufferSlot(0, r.buffers.lines, 0, len(r.lineVerts)*int(unsafe.Sizeof(lineVertex{})))
	p.Draw(len(r.lineVerts), 0)
	return 1
}

func (r *scene3D) drawBillboards(p *wgpu.Pass, samples int) int {
	calls := 0
	for _, additive := range []bool{false, true} {
		from, to := 0, r.billAdd
		if additive {
			from, to = r.billAdd, len(r.billVerts)
		}
		if to == from {
			continue
		}
		pl := r.pipeline(fmt.Sprintf("billboards a%v s%d", additive, samples), func() wgpu.PipelineSpec {
			b := wgpu.Blend{
				Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.OneMinusSrcAlpha},
				Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.OneMinusSrcAlpha},
			}
			if additive {
				b = wgpu.Blend{
					Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.One},
					Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.One},
				}
			}
			return wgpu.PipelineSpec{
				Layouts:  []*wgpu.BindGroupLayout{r.frameSet},
				Shader:   r.shader,
				Vertex:   "billboard_vs",
				Fragment: "billboard_fs",
				Buffers: []wgpu.VertexBuffer{{Stride: unsafe.Sizeof(billVertex{}), Attributes: []wgpu.Attribute{
					{Location: 0, Format: wgpu.Float32x3, Offset: unsafe.Offsetof(billVertex{}.Center)},
					{Location: 1, Format: wgpu.Float32x3, Offset: unsafe.Offsetof(billVertex{}.Corner)},
					{Location: 2, Format: wgpu.Float32x4, Offset: unsafe.Offsetof(billVertex{}.Color)}}}},
				Format:  sceneColor,
				Samples: samples,
				Blend:   &b,
				Depth:   &wgpu.DepthState{Format: sceneDepth},
			}
		})
		if pl == nil {
			continue
		}
		stride := int(unsafe.Sizeof(billVertex{}))
		p.SetPipeline(pl)
		p.SetVertexBufferSlot(0, r.buffers.bills, from*stride, (to-from)*stride)
		p.Draw(to-from, 0)
		calls++
	}
	return calls
}

// postSlots are where this frame's fullscreen passes find their uniforms.
type postSlots struct {
	prefilter, composite, present int
	down, up                      []int
}

// postUniforms lays out the uniforms of this frame's fullscreen passes.
func (r *scene3D) postUniforms(
	env *render.Environment,
	s *surface3D,
	post *render.Post,
	_ *render.View,
	fb geom.Vec2,
) postSlots {
	var out postSlots
	_, threshold := env.Bloom()
	out.prefilter = uniform(r, postUniforms{Threshold: max(threshold, 0.01)})
	for i := 1; i < len(s.bloom); i++ {
		src := s.bloom[i-1]
		out.down = append(
			out.down,
			uniform(r, postUniforms{Texel: [2]float32{1 / float32(src.Width), 1 / float32(src.Height)}}),
		)
	}
	for i := len(s.bloom) - 1; i > 0; i-- {
		src := s.bloom[i]
		out.up = append(
			out.up,
			uniform(r, postUniforms{Texel: [2]float32{0.5 / float32(src.Width), 0.5 / float32(src.Height)}}),
		)
	}
	bloom, _ := env.Bloom()
	saturation, contrast, shadowTint, highlightTint := env.ColorGrade()
	size := fb
	if !post.Plain() {
		size = geom.V2(float32(s.w), float32(s.h))
	}
	out.composite = uniform(
		r,
		postUniforms{
			BloomStrength: bloom,
			Exposure:      env.Exposure(),
			Vignette:      env.Vignette(),
			Saturation:    saturation,
			Contrast:      contrast,
			ShadowTint:    rgb(shadowTint),
			HighlightTint: rgb(highlightTint),
			Time:          r.time,
			Size:          [2]float32{size.X, size.Y},
		},
	)
	nearest := float32(0)
	if post.Upscale == render.FilterNearest {
		nearest = 1
	}
	out.present = uniform(r, postUniforms{Nearest: nearest, Size: [2]float32{float32(s.w), float32(s.h)}})
	return out
}

func (r *scene3D) postGroup(at int, src, aux *wgpu.Texture) *wgpu.BindGroup {
	return r.group(
		r.postSet,
		wgpu.BindEntry{Binding: 0, Buffer: r.buffers.uniforms, Offset: at, Size: int(unsafe.Sizeof(postUniforms{}))},
		wgpu.BindEntry{Binding: 1, View: src.View()},
		wgpu.BindEntry{Binding: 2, Sampler: r.linear},
		wgpu.BindEntry{Binding: 3, View: aux.View()},
		wgpu.BindEntry{Binding: 4, Sampler: r.linear},
	)
}

// fullscreen runs pipeline over into, reading group, keeping what into
// holds when additive.
func fullscreen(e *wgpu.Encoder, into *wgpu.Texture, pl *wgpu.RenderPipeline, g *wgpu.BindGroup, keep bool) {
	var clearTo *wgpu.Color
	if !keep {
		clearTo = &wgpu.Color{}
	}
	p := e.BeginPass(into.View(), clearTo)
	if pl != nil {
		p.SetPipeline(pl)
		p.SetBindGroup(0, g)
		p.Draw(3, 0)
	}
	p.End()
}

func (r *scene3D) bloomPasses(e *wgpu.Encoder, s *surface3D, slots postSlots) int {
	if len(s.bloom) == 0 {
		return 0
	}
	fullscreen(
		e,
		s.bloom[0],
		r.postPipeline("prefilter_fs", sceneColor, false),
		r.postGroup(slots.prefilter, s.resolved, s.resolved),
		false,
	)
	for i := 1; i < len(s.bloom); i++ {
		fullscreen(
			e,
			s.bloom[i],
			r.postPipeline("down_fs", sceneColor, false),
			r.postGroup(slots.down[i-1], s.bloom[i-1], s.bloom[i-1]),
			false,
		)
	}
	for n, i := 0, len(s.bloom)-1; i > 0; n, i = n+1, i-1 {
		fullscreen(
			e,
			s.bloom[i-1],
			r.postPipeline("up_fs", sceneColor, true),
			r.postGroup(slots.up[n], s.bloom[i], s.bloom[i]),
			true,
		)
	}
	return 2 + 2*len(s.bloom)
}

// finish tonemaps the scene onto target, straight when the post chain is
// plain, else through ldr, each effect and the present.
func (r *scene3D) finish(
	e *wgpu.Encoder,
	s *surface3D,
	post *render.Post,
	view *render.View,
	target *wgpu.Texture,
	slots postSlots,
) int {
	bloom := s.resolved
	if len(s.bloom) > 0 {
		bloom = s.bloom[0]
	}
	if post.Plain() {
		fullscreen(
			e,
			target,
			r.postPipeline("composite_fs", target.Format, false),
			r.postGroup(slots.composite, s.resolved, bloom),
			false,
		)
		return 0
	}
	fullscreen(
		e,
		s.ldr[0],
		r.postPipeline("composite_fs", ldrFormat, false),
		r.postGroup(slots.composite, s.resolved, bloom),
		false,
	)
	src, calls := 0, 0
	for _, eff := range post.Active() {
		into := s.ldr[1-src]
		if eff.Screen() {
			into = target
		}
		if !r.ui.runEffect(
			e,
			eff,
			s.ldr[src],
			into,
			s.depthCopy,
			effectScene{size: [2]float32{float32(s.w), float32(s.h)},
				out: [2]float32{float32(into.Width), float32(into.Height)}, time: r.time, depthRange: depthRange(view)},
		) {
			continue
		}
		calls++
		if eff.Screen() {
			return calls
		}
		src = 1 - src
	}
	fullscreen(
		e,
		target,
		r.postPipeline("present_fs", target.Format, false),
		r.postGroup(slots.present, s.ldr[src], s.ldr[src]),
		false,
	)
	return calls + 1
}

func depthRange(view *render.View) [3]float32 {
	ortho := float32(0)
	if view.Projection != nil && view.Projection.OrthoHeight > 0 {
		ortho = 1
	}
	return [3]float32{view.Near, view.Far, ortho}
}

// copyDepth takes the depth's first sample into a float texture effects
// can read.
func (r *scene3D) copyDepth(e *wgpu.Encoder, s *surface3D) {
	if s.depthCopy == nil {
		s.depthCopy = r.dev.CreateTexture(s.w, s.h, wgpu.R32Float, wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding)
	}
	layout, entry, binding := r.copySet, "copy_ms_fs", 0
	if s.samples <= 1 {
		layout, entry, binding = r.copyOneSet, "copy_fs", 1
	}
	pl := r.pipeline("depth copy "+entry, func() wgpu.PipelineSpec {
		return wgpu.PipelineSpec{
			Layouts:  []*wgpu.BindGroupLayout{layout},
			Shader:   r.depthCopy,
			Vertex:   fullVS,
			Fragment: entry,
			Format:   wgpu.R32Float,
		}
	})
	fullscreen(e, s.depthCopy, pl, r.group(layout, wgpu.BindEntry{Binding: binding, View: s.depth.View()}), false)
}

// submitted lets go of this frame's bind groups.
func (r *scene3D) submitted() {
	for _, f := range r.release {
		f()
	}
	r.release = r.release[:0]
}

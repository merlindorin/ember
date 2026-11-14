package gpu

import (
	"fmt"
	"image"
	imagedraw "image/draw"
	"math"
	"slices"
	"time"
	"unsafe"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/errlog"
	"github.com/lantervale/ember/internal/wgpu"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/world"
)

// webgpu2D draws the 2D lists through WebGPU, into the texture the desktop
// runner gives each window this frame: the window's background, what
// negative layers drew, then the rest.
type webgpu2D struct {
	dev *wgpu.Device
	// shaders are the 2D shader reading the light picture alone, and
	// shaded (see uiShader).
	shaders   [2]*wgpu.Shader
	drawSet   *wgpu.BindGroupLayout
	textures  *wgpu.BindGroupLayout
	pipelines map[pipelineKey]*wgpu.RenderPipeline
	groups    map[groupKey]*wgpu.BindGroup
	samplers  [2][2]*wgpu.Sampler

	uniforms, vertices, indices *wgpu.Buffer
	drawGroup                   *wgpu.BindGroup

	atlases map[*draw.Font]gpuTexture
	images  map[*render.Texture]gpuTexture
	// targets are the textures passes draw into, by the texture their
	// pictures show, kept while their window draws them.
	targets map[*render.Texture]*passTarget
	effects map[*render.Effect]*webgpuEffect
	// scene draws the 3D view of a window that has one.
	scene *scene3D
	// timers time each window's passes while F3 or EMBER_STATS asks.
	timers map[*platform.Window]*wgpu.Timers
	blank  *wgpu.Texture
	alpha  *wgpu.Texture
	// farDepth is the depth an effect without a scene reads: far, 1.
	farDepth *wgpu.Texture
	lights   lighting

	// slots are this window's uniforms, one uniformStride apart.
	slots []byte
	// stage gathers lists written together (writeAll).
	stage []byte
	stats *world.Stats
	frame uint64
	err   error
}

// uniformStride is how far apart one batch's uniforms are: WebGPU's
// minimum uniform offset alignment.
const uniformStride = 256

// drawUniforms is the Draw struct of uiWGSL.
type drawUniforms struct {
	Screen        [2]float32
	Scale, Time   float32
	HDR, Exposure float32
	Masked, Blend float32
	Targets       uint32
	_             [3]uint32
	// Round and Radii are the batch's rounded clip (decision 0058): its
	// rect in points as x, y, w, h, zero w for none, and its top-left,
	// top-right, bottom-right and bottom-left radii.
	Round, Radii [4]float32
}

type gpuTexture struct {
	gpu     *wgpu.Texture
	version uint64
}

// passTarget is the texture a pass draws into, in its window.
type passTarget struct {
	gpu, spare *wgpu.Texture
	win        *platform.Window
	used       bool
}

func (t *passTarget) release() {
	t.gpu.Release()
	if t.spare != nil {
		t.spare.Release()
	}
}

// passFormat is what passes draw into.
const passFormat = wgpu.RGBA8Unorm

// pipelineKey is what a pipeline differs by: the batch's blend, and the
// format of what it draws into, its samples, and whether it has a depth
// attachment (the 3D scene's, which 2D draws over without testing).
type pipelineKey struct {
	blend   draw.Blend
	format  wgpu.TextureFormat
	samples int
	depth   bool
	shaded  bool
}

// target2D is what a list draws into: its format, samples and depth, and
// the exposure of a 3D scene's HDR colours, 0 for plain colours.
type target2D struct {
	format   wgpu.TextureFormat
	samples  int
	depth    bool
	exposure float32
	// linear writes light, not colours: the light picture's glows.
	linear bool
}

// groupKey is the textures and samplers of a batch's bind group: the
// atlas, the mask, then the images.
type groupKey struct {
	views    [2 + draw.MaxImages]wgpu.TextureView
	samplers [2 + draw.MaxImages]*wgpu.Sampler
}

// blendOf is the blend state of mode m: the target's alpha accumulating as with Alpha, and the shader writing
// Multiply's colour mixed towards white and Screen's and Lighten's
// premultiplied.
func blendOf(m draw.Blend) wgpu.Blend {
	alpha := wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.OneMinusSrcAlpha}
	switch m {
	case draw.Lighten:
		return wgpu.Blend{Color: wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One}, Alpha: alpha}
	case draw.Occlude:
		maxed := wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One}
		return wgpu.Blend{Color: maxed, Alpha: maxed}
	case draw.Add:
		return wgpu.Blend{Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.One}, Alpha: alpha}
	case draw.Multiply:
		return wgpu.Blend{
			Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.Src},
			Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.One},
		}
	case draw.Screen:
		return wgpu.Blend{Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.OneMinusSrc}, Alpha: alpha}
	case draw.Alpha:
	}
	return wgpu.Blend{
		Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.OneMinusSrcAlpha},
		Alpha: alpha,
	}
}

func (r *webgpu2D) init(d *wgpu.Device) error {
	r.dev = d
	for i, shaded := range [2]bool{false, true} {
		sh, err := d.CreateShader("ui", uiShader(shaded))
		if err != nil {
			return err
		}
		r.shaders[i] = sh
	}
	var err error
	if r.drawSet, err = d.CreateBindGroupLayout(
		wgpu.LayoutEntry{
			Binding:    0,
			Visibility: wgpu.StageVertex | wgpu.StageFragment,
			Kind:       wgpu.BindUniform,
			Dynamic:    true,
		},
	); err != nil {
		return err
	}
	var entries []wgpu.LayoutEntry
	for i := range 2 + draw.MaxImages {
		entries = append(entries,
			wgpu.LayoutEntry{Binding: i, Visibility: wgpu.StageFragment, Kind: wgpu.BindTexture},
			wgpu.LayoutEntry{Binding: 10 + i, Visibility: wgpu.StageFragment, Kind: wgpu.BindSampler})
	}
	if r.textures, err = d.CreateBindGroupLayout(entries...); err != nil {
		return err
	}
	r.pipelines = map[pipelineKey]*wgpu.RenderPipeline{}
	r.groups = map[groupKey]*wgpu.BindGroup{}
	r.atlases = map[*draw.Font]gpuTexture{}
	r.images = map[*render.Texture]gpuTexture{}
	r.targets = map[*render.Texture]*passTarget{}
	for i, nearest := range [2]bool{false, true} {
		for j, repeat := range [2]bool{false, true} {
			r.samplers[i][j] = d.CreateSampler(nearest, repeat)
		}
	}
	r.blank = d.CreateTexture(1, 1, wgpu.RGBA8UnormSrgb, wgpu.UsageTextureBinding|wgpu.UsageCopyDst)
	d.WriteTexture(r.blank, []byte{255, 255, 255, 255}, 4)
	r.alpha = d.CreateTexture(1, 1, wgpu.R8Unorm, wgpu.UsageTextureBinding|wgpu.UsageCopyDst)
	d.WriteTexture(r.alpha, []byte{255}, 1)
	r.farDepth = d.CreateTexture(1, 1, wgpu.R32Float, wgpu.UsageTextureBinding|wgpu.UsageCopyDst)
	d.WriteTexture(r.farDepth, []byte{0, 0, 0x80, 0x3f}, 4)
	return r.initLights(d)
}

// pipeline is the pipeline drawing in blend mode m into t, made the first
// time.
func (r *webgpu2D) pipeline(m draw.Blend, t target2D) (*wgpu.RenderPipeline, error) {
	shaded := r.lights.shading
	k := pipelineKey{m, t.format, max(t.samples, 1), t.depth, shaded}
	if p, ok := r.pipelines[k]; ok {
		return p, nil
	}
	var v draw.Vertex
	b := blendOf(m)
	p, err := r.dev.CreateRenderPipeline(wgpu.PipelineSpec{
		Label: fmt.Sprintf("ui blend %d", m), Layouts: []*wgpu.BindGroupLayout{r.drawSet, r.textures, r.lights.readSet},
		Shader: r.shaders[boolIndex(shaded)], Vertex: "vs", Fragment: "fs", Stride: unsafe.Sizeof(v),
		Attributes: []wgpu.Attribute{
			{Location: 0, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(v.Pos)},
			{Location: 1, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(v.UV)},
			{Location: 2, Format: wgpu.Float32x4, Offset: unsafe.Offsetof(v.Color)},
			{Location: 3, Format: wgpu.Float32x4, Offset: unsafe.Offsetof(v.Shape)},
			{Location: 4, Format: wgpu.Float32, Offset: unsafe.Offsetof(v.Extra)},
			{Location: 5, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(v.LightUV)},
			{Location: 6, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(v.LightSpread)},
		},
		Format: t.format, Blend: &b, Samples: t.samples, Depth: depthOver(t.depth),
	})
	if err != nil {
		return nil, err
	}
	r.pipelines[k] = p
	return p, nil
}

func boolIndex(b bool) int {
	if b {
		return 1
	}
	return 0
}

// depthOver is the depth state of 2D drawn over a scene's depth: never
// tested, never written.
func depthOver(depth bool) *wgpu.DepthState {
	if !depth {
		return nil
	}
	return &wgpu.DepthState{Format: sceneDepth, Always: true}
}

// sampler is how t is read: nearest or linear, clamped or repeating.
func (r *webgpu2D) sampler(t *render.Texture) *wgpu.Sampler {
	nearest := t != nil && t.Filter == render.FilterNearest
	repeat := t != nil && t.Repeat
	return r.samplers[b2i(nearest)][b2i(repeat)]
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// image is t on the GPU: the texture a pass draws into when t shows a
// pass, else t's image, uploaded again when its version changes; the
// blank texture for none.
func (r *webgpu2D) image(t *render.Texture) *wgpu.Texture {
	if p, ok := r.targets[t]; ok {
		return p.gpu
	}
	if g, ok := r.lights.glows[t]; ok {
		return g.texture
	}
	if t == nil || t.Image == nil {
		return r.blank
	}
	c, ok := r.images[t]
	if ok && c.version == t.Version() {
		return c.gpu
	}
	if ok && r.patch(t, c.gpu) {
		r.images[t] = gpuTexture{gpu: c.gpu, version: t.Version()}
		return c.gpu
	}
	if ok {
		r.forgetGroups()
		c.gpu.Release()
	}
	rgba, isRGBA := t.Image.(*image.RGBA)
	if !isRGBA || rgba.Rect.Min != (image.Point{}) {
		b := t.Image.Bounds()
		rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		imagedraw.Draw(rgba, rgba.Bounds(), t.Image, b.Min, imagedraw.Src)
	}
	format := wgpu.RGBA8UnormSrgb
	if t.Encoded {
		format = wgpu.RGBA8Unorm
	}
	g := r.dev.CreateTexture(rgba.Rect.Dx(), rgba.Rect.Dy(), format, wgpu.UsageTextureBinding|wgpu.UsageCopyDst)
	if rects, whole := t.Changes(); !whole && len(rects) > 0 && r.patch(t, g) {
		r.images[t] = gpuTexture{gpu: g, version: t.Version()}
		return g
	}
	r.dev.WriteTexture(g, rgba.Pix, rgba.Stride)
	r.images[t] = gpuTexture{gpu: g, version: t.Version()}
	t.Uploaded()
	return g
}

// patch uploads the rectangles of t touched since its last upload into
// g, its texture on the GPU, and reports whether it could: t's image is
// an RGBA of g's size, and only rectangles changed. A texture only ever
// touched by rectangles, such as a tiled picture's page, is never
// uploaded whole: what was never painted stays clear.
func (r *webgpu2D) patch(t *render.Texture, g *wgpu.Texture) bool {
	rects, whole := t.Changes()
	rgba, isRGBA := t.Image.(*image.RGBA)
	if whole || len(rects) == 0 || !isRGBA || rgba.Rect.Min != (image.Point{}) || rgba.Rect.Dx() != g.Width ||
		rgba.Rect.Dy() != g.Height {
		return false
	}
	for _, rect := range rects {
		rect = rect.Intersect(rgba.Rect)
		if rect.Empty() {
			continue
		}
		at := rect.Min.Y*rgba.Stride + rect.Min.X*4
		end := (rect.Max.Y-1)*rgba.Stride + rect.Max.X*4
		r.dev.WriteTextureRegion(g, rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy(), rgba.Pix[at:end], rgba.Stride)
	}
	t.Uploaded()
	return true
}

// atlas is f's glyph atlas on the GPU, uploaded again when f adds glyphs.
func (r *webgpu2D) atlas(f *draw.Font) *wgpu.Texture {
	if f == nil {
		return r.alpha
	}
	c, ok := r.atlases[f]
	if ok && c.version == f.Version() {
		return c.gpu
	}
	if ok {
		r.forgetGroups()
		c.gpu.Release()
	}
	a := f.Atlas
	g := r.dev.CreateTexture(a.Rect.Dx(), a.Rect.Dy(), wgpu.R8Unorm, wgpu.UsageTextureBinding|wgpu.UsageCopyDst)
	r.dev.WriteTexture(g, a.Pix, a.Stride)
	r.atlases[f] = gpuTexture{gpu: g, version: f.Version()}
	return g
}

// forgetGroups lets every texture bind group go, when a texture they hold
// is replaced.
func (r *webgpu2D) forgetGroups() {
	for k, g := range r.lights.pictureGroups {
		g.Release()
		delete(r.lights.pictureGroups, k)
	}
	for k, g := range r.groups {
		g.Release()
		delete(r.groups, k)
	}
}

// group is the bind group of b's textures, made the first time, and the
// bits of its images that are passes' targets.
func (r *webgpu2D) group(b draw.Batch) (*wgpu.BindGroup, uint32) {
	var k groupKey
	k.views[0], k.samplers[0] = r.atlas(b.Atlas).View(), r.samplers[0][0]
	k.views[1], k.samplers[1] = r.blank.View(), r.samplers[0][0]
	if b.Mask != nil {
		k.views[1], k.samplers[1] = r.image(b.Mask).View(), r.sampler(b.Mask)
	}
	for i := range draw.MaxImages {
		k.views[2+i], k.samplers[2+i] = r.blank.View(), r.samplers[0][0]
		if i < b.ImageCount {
			k.views[2+i], k.samplers[2+i] = r.image(b.Images[i]).View(), r.sampler(b.Images[i])
		}
	}
	targets := uint32(0)
	for i := range b.ImageCount {
		_, pass := r.targets[b.Images[i]]
		_, glow := r.lights.glows[b.Images[i]]
		if pass || glow || b.Images[i] != nil && b.Images[i].Encoded {
			targets |= 1 << i
		}
	}
	if g, ok := r.groups[k]; ok {
		return g, targets
	}
	var entries []wgpu.BindEntry
	for i := range k.views {
		entries = append(
			entries,
			wgpu.BindEntry{Binding: i, View: k.views[i]},
			wgpu.BindEntry{Binding: 10 + i, Sampler: k.samplers[i]},
		)
	}
	g := r.dev.CreateBindGroup(r.textures, entries...)
	r.groups[k] = g
	return g, targets
}

// span is where one list's vertices, indices and batch uniforms go in this
// window's buffers.
type span struct {
	list                    *draw.List
	vertices, indices, slot int
	// written is set once the list is in the buffers, written with
	// others by writeAll, so record leaves it.
	written bool
}

// writeAll writes the lists of spans into the buffers in one write each
// for their vertices and their indices, through stage, and marks them
// written: many small lists, such as silhouettes, cost one call. A few
// are left to record, which writes each straight from its list.
func (r *webgpu2D) writeAll(spans []*span) {
	if len(spans) <= 4 {
		return
	}
	vsize := int(unsafe.Sizeof(draw.Vertex{}))
	v0, v1, i0, i1 := spans[0].vertices, 0, spans[0].indices, 0
	for _, s := range spans {
		v0, i0 = min(v0, s.vertices), min(i0, s.indices)
		v1, i1 = max(v1, s.vertices+len(s.list.Vertices)*vsize), max(i1, s.indices+len(s.list.Indices)*4)
	}
	r.stage = slices.Grow(r.stage[:0], max(v1-v0, i1-i0))[:max(v1-v0, i1-i0)]
	for _, s := range spans {
		copy(
			r.stage[s.vertices-v0:],
			unsafe.Slice((*byte)(unsafe.Pointer(&s.list.Vertices[0])), len(s.list.Vertices)*vsize),
		)
	}
	r.dev.WriteBuffer(r.vertices, v0, unsafe.Pointer(&r.stage[0]), v1-v0)
	for _, s := range spans {
		copy(r.stage[s.indices-i0:], unsafe.Slice((*byte)(unsafe.Pointer(&s.list.Indices[0])), len(s.list.Indices)*4))
	}
	r.dev.WriteBuffer(r.indices, i0, unsafe.Pointer(&r.stage[0]), i1-i0)
	for _, s := range spans {
		s.written = true
	}
}

// plan places lists one after another in the buffers, and grows the
// buffers to hold them all, before anything is recorded: a write lands
// before the submit that reads it, so lists of one submit never share a
// range.
func (r *webgpu2D) plan(lists []*draw.List) []span {
	var spans []span
	v, i, slots := 0, 0, 0
	for _, c := range lists {
		if c == nil || len(c.Vertices) == 0 || len(c.Indices) == 0 {
			continue
		}
		spans = append(spans, span{list: c, vertices: v, indices: i, slot: slots})
		v += align(len(c.Vertices)*int(unsafe.Sizeof(draw.Vertex{})), 256)
		i += align(len(c.Indices)*4, 256)
		slots += len(c.Batches)
	}
	r.vertices = r.grow(r.vertices, v, wgpu.BufferVertex|wgpu.BufferCopyDst)
	r.indices = r.grow(r.indices, i, wgpu.BufferIndex|wgpu.BufferCopyDst)
	need := max(slots, 1) * uniformStride
	if r.uniforms == nil || r.uniforms.Size < need {
		r.uniforms = r.grow(r.uniforms, need, wgpu.BufferUniform|wgpu.BufferCopyDst)
		if r.drawGroup != nil {
			r.drawGroup.Release()
		}
		r.drawGroup = r.dev.CreateBindGroup(
			r.drawSet,
			wgpu.BindEntry{Binding: 0, Buffer: r.uniforms, Size: int(unsafe.Sizeof(drawUniforms{}))},
		)
	}
	if cap(r.slots) < slots*uniformStride {
		r.slots = make([]byte, slots*uniformStride)
	}
	r.slots = r.slots[:slots*uniformStride]
	return spans
}

// grow is b, or, when b is nil or holds fewer than size bytes, a new
// buffer at least twice as big, so it seldom grows.
func (r *webgpu2D) grow(b *wgpu.Buffer, size int, usage wgpu.Usage) *wgpu.Buffer {
	size = max(size, 4096)
	if b != nil && b.Size >= size {
		return b
	}
	if b != nil {
		size = max(size, 2*b.Size)
		b.Release()
	}
	return r.dev.CreateBuffer(align(size, 256), usage)
}

func align(n, to int) int { return (n + to - 1) / to * to }

// record draws s's list into pass, in size points over fb pixels of
// format: each batch with its blend's pipeline, its textures, its
// uniforms and its clip. Into a pass, it leaves out what would show the
// pass in itself.
func (r *webgpu2D) record(pass *wgpu.Pass, s span, size, fb geom.Vec2, t target2D, into *render.Texture) {
	c := s.list
	vbytes, ibytes := len(c.Vertices)*int(unsafe.Sizeof(draw.Vertex{})), len(c.Indices)*4
	if !s.written {
		r.dev.WriteBuffer(r.vertices, s.vertices, unsafe.Pointer(&c.Vertices[0]), vbytes)
		r.dev.WriteBuffer(r.indices, s.indices, unsafe.Pointer(&c.Indices[0]), ibytes)
	}
	pass.SetVertexBuffer(r.vertices, s.vertices, vbytes)
	pass.SetIndexBuffer(r.indices, s.indices, ibytes)
	sx, sy := fb.X/max(size.X, 1), fb.Y/max(size.Y, 1)
	lightBound := false
	var current *wgpu.RenderPipeline
	var bound *wgpu.BindGroup
	for n, b := range c.Batches {
		if b.Count == 0 || (into != nil && (b.Image() == into || b.Mask == into)) {
			continue
		}
		x, y, w, h := scissor(&b, fb, sx, sy)
		if w <= 0 || h <= 0 {
			continue
		}
		p, err := r.pipeline(b.Blend, t)
		if err != nil {
			r.fail(err)
			return
		}
		if p != current {
			pass.SetPipeline(p)
			current = p
		}
		if !lightBound {
			pass.SetBindGroup(2, r.lights.current)
			lightBound = true
		}
		g, targets := r.group(b)
		at := (s.slot + n) * uniformStride
		*(*drawUniforms)(unsafe.Pointer(&r.slots[at])) = batchUniforms(&b, t, size, sx, c.Time(), targets)
		pass.SetBindGroup(0, r.drawGroup, uint32(at)) //nolint:gosec // a slot's offset, within the uniform buffer
		if g != bound {
			pass.SetBindGroup(1, g)
			bound = g
			if r.stats != nil {
				r.stats.Binds2D++
			}
		}
		pass.SetScissor(x, y, w, h)
		pass.DrawIndexed(b.Count, b.First)
		if r.stats != nil {
			r.stats.Draws2D++
			r.stats.Vertices2D += b.Count
		}
	}
}

// scissor is the part of the framebuffer fb that batch b draws in, its
// clip scaled by sx, sy.
func scissor(b *draw.Batch, fb geom.Vec2, sx, sy float32) (int, int, int, int) {
	if !b.Clipped {
		return 0, 0, int(fb.X), int(fb.Y)
	}
	x0, y0 := clampTo(b.Clip.X*sx, fb.X), clampTo(b.Clip.Y*sy, fb.Y)
	x1, y1 := clampTo((b.Clip.X+b.Clip.W)*sx, fb.X), clampTo((b.Clip.Y+b.Clip.H)*sy, fb.Y)
	return x0, y0, x1 - x0, y1 - y0
}

// batchUniforms are batch b's uniforms drawn into t, for a screen of size
// points scaled by sx.
func batchUniforms(b *draw.Batch, t target2D, size geom.Vec2, sx, time float32, targets uint32) drawUniforms {
	masked := float32(0)
	if b.Mask != nil {
		masked = 1
	}
	hdr, exposure := float32(0), float32(1)
	if t.exposure > 0 {
		hdr, exposure = 1, t.exposure
	}
	if t.linear {
		hdr = 2
	}
	return drawUniforms{
		Screen:   [2]float32{max(size.X, 1), max(size.Y, 1)},
		Scale:    sx,
		Time:     time,
		HDR:      hdr,
		Exposure: exposure,
		Masked:   masked,
		Blend:    float32(b.Blend),
		Targets:  targets,
		Round:    [4]float32{b.Round.Rect.X, b.Round.Rect.Y, b.Round.Rect.W, b.Round.Rect.H},
		Radii:    b.Round.Radii,
	}
}

// clampTo is v in [0, limit], to the nearest pixel.
func clampTo(v, limit float32) int { return int(math.Round(float64(min(max(v, 0), limit)))) }

// fail reports what couldn't be made, once.
func (r *webgpu2D) fail(err error) {
	if r.err == nil {
		r.err = err
		errlog.Print("gpu: webgpu: " + err.Error())
	}
}

// count points the counts at w's stats, zeroed once a frame.
func (r *webgpu2D) count(w *world.World) {
	r.stats = w.Stats()
	if f := w.Time().Frame; f != r.frame {
		r.frame = f
		r.stats.Draws2D, r.stats.Binds2D, r.stats.Vertices2D, r.stats.Passes2D = 0, 0, 0, 0
	}
}

func (r *webgpu2D) Present(w *world.World, _ float32) {
	if !wm.Presenting(w) {
		return
	}
	s := wgpu.SharedOf(w)
	win := wm.Target(w)
	if s == nil || win == nil {
		return
	}
	target := s.Target(win)
	if target == nil {
		return
	}
	if r.dev == nil {
		if err := r.init(s.Device); err != nil {
			panic(fmt.Errorf("gpu: the 2D renderer can't start on this GPU: %w", err))
		}
		if r.scene != nil {
			if err := r.scene.init(s.Device); err != nil {
				r.fail(err)
				r.scene = nil
			}
		}
	}
	r.count(w)
	t := r.timer(w, win)
	t.Frame(func(name string, d time.Duration) { w.Stats().Record("gpu "+name, d) })
	r.window(w, win, target, t, draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
	if err := wgpu.TakeError(); err != nil {
		r.fail(err)
	}
}

// window draws win's passes into their textures, then its background,
// back list and front list into target, in one submit.
func (r *webgpu2D) window(
	w *world.World,
	win *platform.Window,
	target *wgpu.Texture,
	timers *wgpu.Timers,
	passes []*draw.Pass,
	back, front *draw.List,
) {
	if li := draw.LightingFor(w, win); li != nil {
		r.bakeStatic(li.Static)
	}
	passes = r.prepare(win, passes)
	lists := make([]*draw.List, 0, len(passes)+3)
	for _, p := range passes {
		lists = append(lists, p.List)
	}
	li := draw.LightingFor(w, win)
	var occluders, emissive, ambient, debug *draw.List
	if li != nil {
		occluders, emissive, ambient, debug = li.Occluders, li.Emissive, li.AmbientMap, li.Debug
	}
	planned := slices.Concat(lists, []*draw.List{back, front, occluders, emissive, ambient, debug})
	if li != nil {
		for _, s := range li.Silhouettes {
			planned = append(planned, s.List)
		}
	}
	spans := r.plan(planned)
	at := func(c *draw.List) (span, bool) {
		for _, s := range spans {
			if s.list == c {
				return s, true
			}
		}
		return span{}, false
	}
	e := r.dev.CreateEncoder()
	spanOf := func(c *draw.List) *span {
		if s, ok := at(c); ok && c != nil {
			return &s
		}
		return nil
	}
	ls := lightSpans{occluders: spanOf(occluders), emissive: spanOf(emissive), ambient: spanOf(ambient)}
	if li != nil {
		var written []*span
		for _, s := range li.Silhouettes {
			sp := spanOf(s.List)
			ls.silhouettes = append(ls.silhouettes, sp)
			if sp != nil {
				written = append(written, sp)
			}
		}
		r.writeAll(written)
	}
	r.light(e, win, li, ls, geom.V2(float32(target.Width), float32(target.Height)), timers)
	r.drawPasses(e, passes, at, timers)
	r.drawWindow(w, e, win, target, back, front, debug, at, timers)
	if len(r.slots) > 0 {
		r.dev.WriteBuffer(r.uniforms, 0, unsafe.Pointer(&r.slots[0]), len(r.slots))
	}
	timers.Resolve(e)
	e.Submit()
	for _, g := range r.lights.debugGroups {
		g.Release()
	}
	r.lights.debugGroups = r.lights.debugGroups[:0]
	timers.Submitted()
	r.lightsSubmitted(win)
	if r.scene != nil {
		r.scene.submitted()
	}
}

// drawPasses draws each of a window's offscreen passes, then what affects
// its picture.
func (r *webgpu2D) drawPasses(
	e *wgpu.Encoder,
	passes []*draw.Pass,
	at func(*draw.List) (span, bool),
	timers *wgpu.Timers,
) {
	for _, p := range passes {
		t := r.targets[p.Target].gpu
		pass := e.BeginPass(t.View(), &wgpu.Color{}, timers.Span("pass "+p.Name))
		if s, ok := at(p.List); ok {
			r.record(
				pass,
				s,
				p.Screen,
				geom.V2(float32(p.Width), float32(p.Height)),
				target2D{format: passFormat},
				p.Target,
			)
		}
		pass.End()
		r.affect(e, r.targets[p.Target], p)
		if r.stats != nil {
			r.stats.Passes2D++
		}
	}
}

// drawWindow draws win onto target: the 3D view and back list, the front
// list, the lighting debug views and the debug list.
func (r *webgpu2D) drawWindow(
	w *world.World,
	e *wgpu.Encoder,
	win *platform.Window,
	target *wgpu.Texture,
	back, front, debug *draw.List,
	at func(*draw.List) (span, bool),
	timers *wgpu.Timers,
) {
	// A 3D view draws the scene, over the back list, onto target; the
	// front list then draws over it.
	behind := []*draw.List{back, front}
	clearTo := &wgpu.Color{}
	bg := win.Background().SRGB()
	*clearTo = wgpu.Color{R: float64(bg[0]), G: float64(bg[1]), B: float64(bg[2]), A: 1}
	if r.scene != nil && r.scene.active(win) {
		var backSpan *span
		if s, ok := at(back); ok {
			backSpan = &s
		}
		if r.scene.draw(w, e, win, target, backSpan, timers) {
			behind, clearTo = []*draw.List{front}, nil
		}
	}
	pass := e.BeginPass(target.View(), clearTo, timers.Span("2d window"))
	fb := geom.V2(float32(target.Width), float32(target.Height))
	for _, c := range behind {
		if s, ok := at(c); ok {
			r.record(pass, s, win.Size(), fb, target2D{format: target.Format}, nil)
		}
	}
	r.lightDebug(w, pass, win, target.Format, fb)
	if s, ok := at(debug); ok && debug != nil {
		r.record(pass, s, win.Size(), fb, target2D{format: target.Format}, nil)
	}
	pass.End()
}

// timer is win's GPU timers, timing one frame in gpuSample while F3
// shows, every frame under EMBER_STATS.
func (r *webgpu2D) timer(w *world.World, win *platform.Window) *wgpu.Timers {
	if r.timers == nil {
		r.timers = map[*platform.Window]*wgpu.Timers{}
	}
	t := r.timers[win]
	if t == nil {
		t = r.dev.NewTimers()
		r.timers[win] = t
	}
	t.On = timing(w)
	return t
}

// prepare gives each of win's passes a texture of its size, and lets go of
// the textures of passes win no longer draws; it gives the passes there
// is something to draw for.
func (r *webgpu2D) prepare(win *platform.Window, passes []*draw.Pass) []*draw.Pass {
	out := passes[:0:0]
	for _, p := range passes {
		if p.Width <= 0 || p.Height <= 0 {
			continue
		}
		t := r.targets[p.Target]
		if t != nil && (t.gpu.Width != p.Width || t.gpu.Height != p.Height) {
			r.forgetGroups()
			t.release()
			t = nil
		}
		if t == nil {
			t = &passTarget{
				gpu: r.dev.CreateTexture(
					p.Width,
					p.Height,
					passFormat,
					wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding|wgpu.UsageCopySrc,
				),
				win: win,
			}
			r.targets[p.Target] = t
		}
		t.used = true
		out = append(out, p)
	}
	for tex, t := range r.targets {
		if t.win != win {
			continue
		}
		if !t.used {
			r.forgetGroups()
			t.release()
			delete(r.targets, tex)
		}
		t.used = false
	}
	return out
}

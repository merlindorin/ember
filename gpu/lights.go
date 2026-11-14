package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strings"
	"unsafe"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/wgpu"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/world"
)

// The formats of a window's lighting: the occluder picture, a colour
// channel a group of occluders, alpha their height; the shadow maps, two
// shadowed stretches of a ray a texel; a
// directional light's shadow picture; the light picture, light that may
// pass 1 before it is capped.
const (
	occluderFormat = passFormat
	shadowFormat   = wgpu.R32Float
	// stretchFormat is the shadow maps' when a light has an Elevation or
	// a fade: two shadowed stretches a texel.
	stretchFormat = wgpu.RGBA32Float
	sunFormat     = wgpu.R8Unorm
	lightFormat   = wgpu.RGBA16Float
)

// stretching is shader src for shadow maps of two stretches a texel, or
// of one distance.
func stretching(src string, two bool) string {
	if two {
		return src
	}
	return strings.Replace(src, "const STRETCHES = true;", "const STRETCHES = false;", 1)
}

// noShadow is a shadow map's texel where no shadow falls: both stretches
// starting past any radius.
const noShadow = 9

// lightScale is the light picture's resolution, as a share of its
// window's pixels.
const lightScale = draw.LightScale

// maxSuns is how many directional lights a window draws.
const maxSuns = 4

// gpuLight is GPULight of lightPictureWGSL.
type gpuLight struct {
	A, Color, Cone, Shadow, Shape, Size, Height, Tint [4]float32
}

// lightVertex is a corner of a light's quad in the light picture: where it
// is, its offset from the light in points, and which light.
type lightVertex struct {
	NDC, Local [2]float32
	Index, _   float32
}

// lightPictures are a window's lighting textures: what blocks light, the
// shadow maps, the directional lights' shadow pictures and the light
// picture, and the bind groups reading them.
type lightPictures struct {
	occluders, light *wgpu.Texture
	suns             [maxSuns]*wgpu.Texture
	// shadows are the shadow maps, plain (a distance a texel) and with
	// stretches (two a texel), and the groups writing and reading them.
	shadows                   [2]*wgpu.Texture
	shadowGroup, pictureGroup [2]*wgpu.BindGroup
	// probes is where the probes' light is read into, a texel each, and
	// probeGroup reads the light picture for it.
	probes                     *wgpu.Texture
	probeGroup                 *wgpu.BindGroup
	sunShadowGroups, sunGroups [maxSuns]*wgpu.BindGroup
	// mask is where the silhouettes cast from one light fall, and
	// maskGroup reads it.
	mask      *wgpu.Texture
	maskGroup *wgpu.BindGroup
	// atlas is the silhouette mask atlas's pages, and atlasGroups read
	// them.
	atlas       []*wgpu.Texture
	atlasGroups []*wgpu.BindGroup
	// read is the light picture as the 2D shader reads it, with the
	// shaded ones; plain is the light picture alone, as a static bake's
	// copy reads it.
	read, plain *wgpu.BindGroup
	// shaded are the light pictures of drawings ShadedBy fewer occluder
	// groups, made when first needed, and the groups drawing them;
	// shadeMap, in shadeIndex, says for each set of groups left out
	// which picture the 2D shader reads, 0 for the light picture.
	shaded               [maxShades]*wgpu.Texture
	shadeShadows         [2]*wgpu.Texture
	shadeShadowGroup     [maxShades][2]*wgpu.BindGroup
	shadePictureGroup    [maxShades][2]*wgpu.BindGroup
	shadeSunShadowGroups [maxShades][maxSuns]*wgpu.BindGroup
	shadeMap             *wgpu.Buffer
	// exposure holds the light's scale the 2D shader reads.
	exposure   *wgpu.Buffer
	shadeIndex []float32
}

// subtractBlend takes what a light draws out of a light picture.
func subtractBlend() wgpu.Blend {
	return wgpu.Blend{
		Color: wgpu.BlendComponent{Op: wgpu.ReverseSubtract, Src: wgpu.One, Dst: wgpu.One},
		Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.One},
	}
}

// silhouetted is a light shader whose light is drawn times a silhouette
// mask's alpha, at group 1: the part of it a silhouette shadows, what it
// lets through of its ShadowColor kept. It panics when src holds neither
// a light's nor a directional light's return to change, so a reworded
// shader can't silently lose its variant.
func silhouetted(src string) string {
	before := src
	src = strings.Replace(
		src,
		"fn light_fs(in: LightOut) -> @location(0) vec4f {\n\tlet l = lights.at[in.index];",
		"fn light_fs(in: LightOut) -> @location(0) vec4f {\n\tlet l = lights.at[in.index];\n"+
			"\tlet shadowed = textureLoad(silhouette, vec2i(in.pos.xy + l.size.zw), 0).a;\n"+
			"\tif (shadowed <= 0.0) {\n\t\treturn vec4f(0.0);\n\t}",
		1,
	)
	src = strings.Replace(src, "return vec4f(l.color.rgb * f * mix(l.tint.rgb, vec3f(1.0), v), 0.0);",
		"return vec4f(l.color.rgb * f * v * (1.0 - l.tint.rgb) * shadowed, 0.0);", 1)
	src = strings.Replace(
		src,
		"return vec4f(sun.color.rgb * mix(sun.tint.rgb, vec3f(1.0), v), 0.0);",
		"return vec4f(sun.color.rgb * v * (1.0 - sun.tint.rgb) * textureLoad(silhouette, vec2i(in.pos.xy), 0).a, 0.0);",
		1,
	)
	if src == before {
		panic("gpu: silhouetted: no light return to change")
	}
	return src + "\n@group(1) @binding(0) var silhouette: texture_2d<f32>;\n"
}

// The entry points the light shaders and the 3D post passes share.
const (
	fullVS  = "full_vs"
	sunVS   = "sun_vs"
	sunFS   = "sun_fs"
	lightVS = "light_vs"
	lightFS = "light_fs"
)

// maskFormat is a silhouette mask's: its alpha where silhouettes shadow.
const maskFormat = wgpu.RGBA8Unorm

// maxShades is how many light pictures a window draws besides its own,
// one a set of occluder groups some drawing is ShadedBy; a drawing past
// them reads the light picture.
const maxShades = draw.MaxPictures

func (p *lightPictures) release() {
	textures := []*wgpu.Texture{p.occluders, p.shadows[0], p.shadows[1], p.light, p.probes, p.mask}
	textures = append(append(append(textures, p.suns[:]...), p.shaded[:]...), p.shadeShadows[:]...)
	for _, t := range textures {
		if t != nil {
			t.Release()
		}
	}
	groups := append(
		[]*wgpu.BindGroup{
			p.shadowGroup[0],
			p.shadowGroup[1],
			p.pictureGroup[0],
			p.pictureGroup[1],
			p.read,
			p.plain,
			p.probeGroup,
			p.maskGroup,
		},
		append(p.sunShadowGroups[:], p.sunGroups[:]...)...)
	for i := range maxShades {
		groups = append(
			append(append(groups, p.shadeShadowGroup[i][:]...), p.shadePictureGroup[i][:]...),
			p.shadeSunShadowGroups[i][:]...)
	}
	for _, g := range groups {
		if g != nil {
			g.Release()
		}
	}
	p.shadeMap.Release()
	p.exposure.Release()
	for i := range p.atlas {
		p.atlas[i].Release()
		p.atlasGroups[i].Release()
	}
}

// lighting is webgpu2D's lighting state.
type lighting struct {
	// readSet is the 2D shader's group 2, the light pictures it reads;
	// unlit reads white, and current is the window's this frame.
	// lightSet is a picture and its sampler.
	readSet, lightSet *wgpu.BindGroupLayout
	unlit, current    *wgpu.BindGroup
	unlitMap          *wgpu.Buffer
	unlitExposure     *wgpu.Buffer
	// exposed is the exposure written, in memory of its own for cgo.
	exposed []float32
	// shading is set while the window drawn reads shaded light pictures,
	// its lists drawn by the shaded 2D shader.
	shading bool
	// shadeRows and shadeSuns are the shadow rows and the directional
	// lights' uniforms of the shaded light pictures, shadeData theirs in
	// memory of their own.
	shadeRows [maxShades]*wgpu.Buffer
	// reach is where each light of the frame reaches, for the shaded
	// pictures, shadeVerts the quads of those redrawn in them and
	// shadeVertBuf where they go; copyPipe copies the light picture into
	// a shaded one and subtractPipe takes a light out of it. redrawn is
	// how many lights the last shaded picture redrew, -1 for all.
	// drawnAs is, for each of the frame's lights, its index among the
	// lights drawn as quads, -2−i for directional light i, or -1 when
	// left out; maskSet reads a silhouette mask, and silhouettePipe and
	// sunSilhouettePipe take a light's part under it out of a picture.
	drawnAs    []int
	casting    []casting
	silVerts   []lightVertex
	silVertBuf *wgpu.Buffer
	// castOf is, for each of the Lighting's Silhouettes, the index of its
	// light among those drawn, -1 when it casts none in view; casts are
	// those lights' tile shifts, elevations and areas the mask shader
	// reads, pageSide a page's side; order, instanceData and buckets are
	// the frame's silhouette instances, and instBuf holds them.
	castOf            []int
	casts             [][4]float32
	castsBuf          *wgpu.Buffer
	pageSide          []float32
	pageBuf           *wgpu.Buffer
	castGroup         *wgpu.BindGroup
	order             []int32
	instanceData      []silInstance
	buckets           []silBucket
	instBuf           *wgpu.Buffer
	silMaskPipe       *wgpu.RenderPipeline
	pictureSetLayout  *wgpu.BindGroupLayout
	pictureGroups     map[wgpu.TextureView]*wgpu.BindGroup
	castSet           *wgpu.BindGroupLayout
	bucketOf          map[silKey]int
	bucketOfQuad      []int
	maskSet           *wgpu.BindGroupLayout
	silhouettePipe    [2]*wgpu.RenderPipeline
	sunSilhouettePipe *wgpu.RenderPipeline
	// silhouetteBackPipe and sunSilhouetteBackPipe add that part back,
	// to a copy of a picture it was taken out of.
	silhouetteBackPipe    [2]*wgpu.RenderPipeline
	sunSilhouetteBackPipe *wgpu.RenderPipeline
	// drawn are the casting lights a silhouettes pass draws, on one page.
	drawn []casting
	// out and outSuns are the lights, by quad and by directional light,
	// a picture being drawn leaves out; sunSubtractPipe takes a
	// directional light out of a picture.
	out, outSuns    []int
	sunSubtractPipe *wgpu.RenderPipeline
	// maskClearPipe clears a light's part of the mask.
	maskClearPipe *wgpu.RenderPipeline
	reach         []lightReach
	reached       []int
	excluded      []geom.Rect
	shadeVerts    []lightVertex
	shadeVertBuf  *wgpu.Buffer
	copyPipe      *wgpu.RenderPipeline
	subtractPipe  [2]*wgpu.RenderPipeline
	redrawn       int
	shadeSuns     [maxShades][maxSuns]*wgpu.Buffer
	shadeData     [][4]float32
	shadowSet     *wgpu.BindGroupLayout
	pictureSet    *wgpu.BindGroupLayout
	sunShadowSet  *wgpu.BindGroupLayout
	sunSet        *wgpu.BindGroupLayout
	shadowPipe    [2]*wgpu.RenderPipeline
	// debugPipes show a lighting texture over a window, F3's F5, one a
	// window format; debugSet is their bind group's layout, debugBuf
	// their uniforms.
	debugPipes map[wgpu.TextureFormat]*wgpu.RenderPipeline
	debugSet   *wgpu.BindGroupLayout
	debugBuf   *wgpu.Buffer
	// debugGroups are the frame's debug bind groups, let go of after it;
	// lastRows and lastStretch what each window's shadow maps held.
	debugGroups []*wgpu.BindGroup
	// static is the static lights\' baked tiles.
	static        staticBakes
	lastRows      map[*platform.Window]int
	lastStretch   map[*platform.Window]int
	picturePipe   [2]*wgpu.RenderPipeline
	sunShadowPipe *wgpu.RenderPipeline
	sunPipe       *wgpu.RenderPipeline
	// rowsBuf, lightsBuf and pointsBuf are the shadow rows, the lights and
	// their shape points; sunParams and sunColors each directional
	// light's.
	rowsBuf, lightsBuf, pointsBuf *wgpu.Buffer
	sunParams, sunColors          [maxSuns]*wgpu.Buffer
	lightVerts                    *wgpu.Buffer
	sampler                       *wgpu.Sampler
	pictures                      map[*platform.Window]*lightPictures
	// rows, lights, points, suns and vertices are this frame's, in memory
	// of their own, as cgo reads them.
	rows     [][4]float32
	lights   []gpuLight
	points   [][4]float32
	suns     [][4]float32
	vertices []lightVertex
	// glows are emissive sprites' glow textures, by the texture standing
	// for them in the draw lists.
	glows    map[*render.Texture]*glowTexture
	glowSet  *wgpu.BindGroupLayout
	glowPipe *wgpu.RenderPipeline
	// probeSet, probePipe and probesBuf read the light at the probes, and
	// readbacks bring each window's answers back; probeUVs are this
	// frame's probes in the light picture.
	probeSet  *wgpu.BindGroupLayout
	probePipe *wgpu.RenderPipeline
	probesBuf *wgpu.Buffer
	probeUVs  [][4]float32
	readbacks map[*platform.Window]*wgpu.Readback
}

// glowTexture is a glow made once: its texture, and the uniforms and bind
// group it was made with.
type glowTexture struct {
	texture *wgpu.Texture
	params  *wgpu.Buffer
	group   *wgpu.BindGroup
	blur    [][8]float32
}

func (r *webgpu2D) initLights(d *wgpu.Device) error {
	b := &gpuBuild{d: d}
	l := &r.lights
	if err := l.initLayouts(b); err != nil {
		return err
	}
	if err := l.initPipes(b); err != nil {
		return err
	}
	return r.initLightBuffers(b)
}

// gpuBuild makes the lights' GPU objects, keeping the first error: past
// it, pipelines are not made.
type gpuBuild struct {
	d   *wgpu.Device
	err error
}

func (b *gpuBuild) layout(entries ...wgpu.LayoutEntry) *wgpu.BindGroupLayout {
	g, e := b.d.CreateBindGroupLayout(entries...)
	if e != nil && b.err == nil {
		b.err = fmt.Errorf("lights: a bind group layout of %d entries: %w", len(entries), e)
	}
	return g
}

func (b *gpuBuild) pipe(label, src string, spec wgpu.PipelineSpec) *wgpu.RenderPipeline {
	if b.err != nil {
		return nil
	}
	sh, e := b.d.CreateShader(label, src)
	if e != nil {
		b.err = fmt.Errorf("lights: the %q shader: %w", label, e)
		return nil
	}
	spec.Label, spec.Shader = label, sh
	p, e := b.d.CreateRenderPipeline(spec)
	if e != nil {
		b.err = fmt.Errorf("lights: the %q pipeline drawing into %s: %w", label, spec.Format, e)
	}
	return p
}

func (b *gpuBuild) uniform(bytes int) *wgpu.Buffer {
	buf, e := b.d.CreateBufferChecked(bytes, wgpu.BufferUniform|wgpu.BufferCopyDst)
	if e != nil && b.err == nil {
		b.err = fmt.Errorf("lights: %w", e)
	}
	return buf
}

// lightAttributes are a lightVertex's attributes.
func lightAttributes() []wgpu.Attribute {
	var v lightVertex
	return []wgpu.Attribute{
		{Location: 0, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(v.NDC)},
		{Location: 1, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(v.Local)},
		{Location: 2, Format: wgpu.Float32, Offset: unsafe.Offsetof(v.Index)},
	}
}

func (l *lighting) initLayouts(b *gpuBuild) error {
	layout := b.layout
	frag := func(binding int, kind wgpu.BindingKind) wgpu.LayoutEntry {
		return wgpu.LayoutEntry{Binding: binding, Visibility: wgpu.StageFragment, Kind: kind}
	}
	l.lightSet = layout(frag(0, wgpu.BindTexture), frag(1, wgpu.BindSampler))
	l.maskSet = layout(frag(0, wgpu.BindTexture))
	both := func(binding int) wgpu.LayoutEntry {
		return wgpu.LayoutEntry{
			Binding:    binding,
			Visibility: wgpu.StageVertex | wgpu.StageFragment,
			Kind:       wgpu.BindUniform,
		}
	}
	l.castSet = layout(both(0), both(1), both(2))
	l.pictureSetLayout = layout(frag(0, wgpu.BindTexture), frag(1, wgpu.BindSampler))
	l.readSet = layout(
		frag(0, wgpu.BindTexture),
		frag(1, wgpu.BindSampler),
		frag(2, wgpu.BindTexture),
		frag(3, wgpu.BindTexture),
		frag(4, wgpu.BindTexture),
		frag(5, wgpu.BindUniform),
		frag(6, wgpu.BindTexture),
		frag(7, wgpu.BindUniform),
	)
	l.shadowSet = layout(frag(0, wgpu.BindUniform), frag(1, wgpu.BindUnfilterable))
	l.pictureSet = layout(
		frag(0, wgpu.BindUnfilterable),
		frag(1, wgpu.BindUniform),
		frag(2, wgpu.BindUniform),
		frag(3, wgpu.BindUniform),
	)
	l.sunShadowSet = layout(frag(0, wgpu.BindUniform), frag(1, wgpu.BindUnfilterable))
	l.sunSet = layout(frag(0, wgpu.BindTexture), frag(1, wgpu.BindSampler), frag(2, wgpu.BindUniform))
	l.glowSet = layout(frag(0, wgpu.BindUniform), frag(1, wgpu.BindUnfilterable))
	l.debugSet = layout(frag(0, wgpu.BindUniform), frag(1, wgpu.BindUnfilterable))
	l.probeSet = layout(frag(0, wgpu.BindUniform), frag(1, wgpu.BindTexture), frag(2, wgpu.BindSampler))
	return b.err
}

func (l *lighting) initPipes(b *gpuBuild) error {
	add := wgpu.Blend{
		Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.One},
		Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.One},
	}
	pipe := b.pipe
	var v lightVertex
	for i, f := range []wgpu.TextureFormat{shadowFormat, stretchFormat} {
		l.shadowPipe[i] = pipe(
			"light shadows",
			stretching(shadowsWGSL, i == 1),
			wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.shadowSet},
				Vertex: fullVS, Fragment: "shadow_fs", Format: f},
		)
	}
	l.sunShadowPipe = pipe(
		"sun shadows",
		sunShadowWGSL,
		wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.sunShadowSet},
			Vertex: fullVS, Fragment: "sun_shadow_fs", Format: sunFormat},
	)
	l.sunPipe = pipe("sun", sunWGSL, wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.sunSet},
		Vertex: sunVS, Fragment: sunFS, Format: lightFormat, Blend: &add})
	l.glowPipe = pipe("light glow", glowWGSL, wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.glowSet},
		Vertex: fullVS, Fragment: "glow_fs", Format: passFormat})
	l.probePipe = pipe("light probes", probesWGSL, wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.probeSet},
		Vertex: fullVS, Fragment: "probe_fs", Format: probeFormat})
	l.initSilhouettePipes(b, &add)
	var si silInstance
	maxed := wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One}
	l.silMaskPipe = pipe(
		"light silhouette mask",
		silhouetteMaskWGSL,
		wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.castSet, l.pictureSetLayout},
			Vertex: "mask_vs", Fragment: "mask_fs",
			Buffers: []wgpu.VertexBuffer{{Stride: unsafe.Sizeof(si), Instance: true, Attributes: []wgpu.Attribute{
				{Location: 0, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(si.Foot)},
				{Location: 1, Format: wgpu.Float32x2, Offset: unsafe.Offsetof(si.Size)},
				{Location: 2, Format: wgpu.Float32x4, Offset: unsafe.Offsetof(si.Rect)},
				{Location: 3, Format: wgpu.Float32x4, Offset: unsafe.Offsetof(si.UV)},
				{Location: 4, Format: wgpu.Float32, Offset: unsafe.Offsetof(si.Light)},
			}}},
			Format: maskFormat, Blend: &wgpu.Blend{Color: maxed, Alpha: maxed}},
	)
	l.maskClearPipe = pipe(
		"light mask clear",
		maskClearWGSL,
		wgpu.PipelineSpec{Vertex: fullVS, Fragment: "clear_fs", Format: maskFormat},
	)
	l.copyPipe = pipe("light copy", lightCopyWGSL, wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.lightSet},
		Vertex: fullVS, Fragment: "copy_fs", Format: lightFormat})
	subtract := subtractBlend()
	for i := range l.subtractPipe {
		l.subtractPipe[i] = pipe(
			"light subtract",
			stretching(lightPictureWGSL, i == 1),
			wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.pictureSet},
				Vertex: lightVS, Fragment: lightFS, Stride: unsafe.Sizeof(v),
				Attributes: lightAttributes(),
				Format:     lightFormat, Blend: &subtract},
		)
	}
	for i := range l.picturePipe {
		l.picturePipe[i] = pipe(
			"light picture",
			stretching(lightPictureWGSL, i == 1),
			wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.pictureSet},
				Vertex: lightVS, Fragment: lightFS, Stride: unsafe.Sizeof(v),
				Attributes: lightAttributes(),
				Format:     lightFormat, Blend: &add},
		)
	}
	return b.err
}

// initSilhouettePipes makes the pipelines drawing silhouette shadows, by
// lights and by the sun, front and back faces.
func (l *lighting) initSilhouettePipes(b *gpuBuild, add *wgpu.Blend) {
	pipe := b.pipe
	var v lightVertex
	for i := range l.silhouettePipe {
		subtracted := subtractBlend()
		for back, blend := range []*wgpu.Blend{&subtracted, add} {
			pl := pipe(
				"light silhouette",
				silhouetted(stretching(lightPictureWGSL, i == 1)),
				wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.pictureSet, l.maskSet},
					Vertex: lightVS, Fragment: lightFS, Stride: unsafe.Sizeof(v),
					Attributes: lightAttributes(),
					Format:     lightFormat, Blend: blend},
			)
			if back == 0 {
				l.silhouettePipe[i] = pl
			} else {
				l.silhouetteBackPipe[i] = pl
			}
		}
	}
	sunSubtract := subtractBlend()
	l.sunSilhouettePipe = pipe(
		"sun silhouette",
		silhouetted(sunWGSL),
		wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.sunSet, l.maskSet},
			Vertex: sunVS, Fragment: sunFS, Format: lightFormat, Blend: &sunSubtract},
	)
	l.sunSubtractPipe = pipe("sun subtract", sunWGSL, wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.sunSet},
		Vertex: sunVS, Fragment: sunFS, Format: lightFormat, Blend: &sunSubtract})
	l.sunSilhouetteBackPipe = pipe(
		"sun silhouette back",
		silhouetted(sunWGSL),
		wgpu.PipelineSpec{Layouts: []*wgpu.BindGroupLayout{l.sunSet, l.maskSet},
			Vertex: sunVS, Fragment: sunFS, Format: lightFormat, Blend: add},
	)
}

func (r *webgpu2D) initLightBuffers(b *gpuBuild) error {
	d, l := b.d, &r.lights
	uniform := b.uniform
	l.rows = make([][4]float32, draw.MaxShadowLights)
	l.lights = make([]gpuLight, max2DLights)
	l.points = make([][4]float32, maxLightPoints)
	l.suns = make([][4]float32, 3*maxSuns)
	l.rowsBuf = uniform(len(l.rows) * 16)
	l.lightsBuf = uniform(len(l.lights) * int(unsafe.Sizeof(gpuLight{})))
	l.pointsBuf = uniform(len(l.points) * 16)
	for i := range maxSuns {
		l.sunParams[i], l.sunColors[i] = uniform(16), uniform(32)
	}
	l.sampler = d.CreateSampler(false, false)
	for i := range maxShades {
		l.shadeRows[i] = uniform(len(l.rows) * 16)
		for j := range maxSuns {
			l.shadeSuns[i][j] = uniform(16)
		}
	}
	l.shadeData = make([][4]float32, len(l.rows)+maxShades*maxSuns)
	l.casts = make([][4]float32, 2*len(l.lights))
	l.castsBuf = uniform(len(l.casts) * 16)
	l.pageSide = make([]float32, 4)
	l.pageBuf = uniform(16)
	if b.err != nil {
		return b.err
	}
	l.castGroup = d.CreateBindGroup(
		l.castSet,
		wgpu.BindEntry{Binding: 0, Buffer: l.lightsBuf, Size: l.lightsBuf.Size},
		wgpu.BindEntry{
			Binding: 1,
			Buffer:  l.castsBuf,
			Size:    l.castsBuf.Size,
		},
		wgpu.BindEntry{Binding: 2, Buffer: l.pageBuf, Size: 16},
	)
	l.pictureGroups = map[wgpu.TextureView]*wgpu.BindGroup{}
	l.bucketOf = map[silKey]int{}
	l.unlitMap = uniform(shadeMapBytes)
	l.unlitExposure = uniform(16)
	if b.err != nil {
		return b.err
	}
	one := []float32{1, 0, 0, 0}
	d.WriteBuffer(l.unlitExposure, 0, unsafe.Pointer(&one[0]), 16)
	blank := r.blank.View()
	l.unlit = d.CreateBindGroup(
		l.readSet,
		wgpu.BindEntry{Binding: 0, View: blank},
		wgpu.BindEntry{Binding: 1, Sampler: l.sampler},
		wgpu.BindEntry{
			Binding: 2,
			View:    blank,
		},
		wgpu.BindEntry{Binding: 3, View: blank},
		wgpu.BindEntry{Binding: 4, View: blank},
		wgpu.BindEntry{Binding: 5, Buffer: l.unlitMap, Size: shadeMapBytes},
		wgpu.BindEntry{Binding: 6, View: blank},
		wgpu.BindEntry{Binding: 7, Buffer: l.unlitExposure, Size: 16},
	)
	l.current = l.unlit
	l.pictures = map[*platform.Window]*lightPictures{}
	l.glows = map[*render.Texture]*glowTexture{}
	l.probeUVs = make([][4]float32, draw.MaxProbes)
	l.probesBuf = uniform(draw.MaxProbes * 16)
	l.readbacks = map[*platform.Window]*wgpu.Readback{}
	return b.err
}

// probeFormat is what the probes' light is read into: 16 bytes a probe.
const probeFormat = wgpu.RGBA32Float

// answer hands back the probes' light that came back for win.
func (r *webgpu2D) answer(win *platform.Window) {
	rb := r.lights.readbacks[win]
	if rb == nil {
		return
	}
	rb.Done(func(rows []byte, _ int, tag any) {
		results, ok := tag.([]*draw.ProbeResult)
		if !ok {
			return
		}
		for i, res := range results {
			px := rows[16*i:]
			f := func(k int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(px[4*k:])) }
			res.Light, res.Ready = color.RGB(f(0), f(1), f(2)), true
		}
	})
}

// probe reads the light picture at li's probes into p's probe texture,
// and has it copied back for them.
func (r *webgpu2D) probe(
	e *wgpu.Encoder,
	win *platform.Window,
	li *draw.Lighting,
	p *lightPictures,
	rect geom.Rect,
	timers *wgpu.Timers,
) {
	l := &r.lights
	n := min(len(li.Probes), draw.MaxProbes)
	if n == 0 {
		return
	}
	rb := l.readbacks[win]
	if rb == nil {
		rb = r.dev.NewReadback()
		l.readbacks[win] = rb
	}
	results := make([]*draw.ProbeResult, n)
	for i, pr := range li.Probes[:n] {
		at := pr.At.Sub(geom.V2(rect.X, rect.Y))
		l.probeUVs[i] = [4]float32{at.X / rect.W, at.Y / rect.H, 0, 0}
		results[i] = pr.Result
	}
	r.dev.WriteBuffer(l.probesBuf, 0, unsafe.Pointer(&l.probeUVs[0]), n*16)
	pass := e.BeginPass(p.probes.View(), &wgpu.Color{}, timers.Span("light probes"))
	pass.SetPipeline(l.probePipe)
	pass.SetBindGroup(0, p.probeGroup)
	pass.SetScissor(0, 0, n, 1)
	pass.Draw(3, 0)
	pass.End()
	rb.Copy(e, p.probes, 16, results)
}

// lightsSubmitted starts reading back what win's probes asked, once its
// frame is submitted.
func (r *webgpu2D) lightsSubmitted(win *platform.Window) {
	if rb := r.lights.readbacks[win]; rb != nil {
		rb.Submitted()
	}
}

// maxGlow is a glow texture's largest side, in pixels.
const maxGlow = 2048

// makeGlows makes the glow textures li's emissive sprites show that
// aren't made yet: each its picture blurred once over its pad.
func (r *webgpu2D) makeGlows(e *wgpu.Encoder, li *draw.Lighting) {
	l := &r.lights
	for stand, g := range li.Glows {
		if _, ok := l.glows[stand]; ok {
			continue
		}
		w := min(texels(g.Region.W+2*g.Pad), maxGlow)
		h := min(texels(g.Region.H+2*g.Pad), maxGlow)
		gt := &glowTexture{
			texture: r.dev.CreateTexture(w, h, passFormat, wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding),
		}
		gt.blur = [][8]float32{{g.Region.X, g.Region.Y, g.Region.W, g.Region.H, g.Pad, max(1, g.Pad/24)}}
		gt.params = r.dev.CreateBuffer(32, wgpu.BufferUniform|wgpu.BufferCopyDst)
		r.dev.WriteBuffer(gt.params, 0, unsafe.Pointer(&gt.blur[0]), 32)
		gt.group = r.dev.CreateBindGroup(l.glowSet, wgpu.BindEntry{Binding: 0, Buffer: gt.params, Size: 32},
			wgpu.BindEntry{Binding: 1, View: r.image(g.Source).View()})
		pass := e.BeginPass(gt.texture.View(), &wgpu.Color{})
		pass.SetPipeline(l.glowPipe)
		pass.SetBindGroup(0, gt.group)
		pass.Draw(3, 0)
		pass.End()
		l.glows[stand] = gt
	}
}

// lightPicturesFor are win's lighting textures for a light rectangle of
// size points over fb pixels, made again when that changes.
func (r *webgpu2D) lightPicturesFor(win *platform.Window, size, fb geom.Vec2) (*lightPictures, error) {
	l := &r.lights
	ow, oh := texels(size.X*draw.OccluderScale), texels(size.Y*draw.OccluderScale)
	lw, lh := texels(fb.X*lightScale), texels(fb.Y*lightScale)
	p := l.pictures[win]
	if p != nil && p.occluders.Width == ow && p.occluders.Height == oh && p.light.Width == lw && p.light.Height == lh {
		return p, nil
	}
	if p != nil {
		p.release()
		delete(l.pictures, win)
	}
	d := r.dev
	var err error
	var made []*wgpu.Texture
	tex := func(w, h int, f wgpu.TextureFormat, use wgpu.Usage) *wgpu.Texture {
		if err != nil {
			return nil
		}
		t, e := d.CreateTextureChecked(wgpu.TextureSpec{Width: w, Height: h, Format: f, Usage: use})
		if e != nil {
			err = fmt.Errorf("lights: a window's light pictures for %v points over %v pixels: %w", size, fb, e)
			return nil
		}
		made = append(made, t)
		return t
	}
	buffer := func(bytes int) *wgpu.Buffer {
		if err != nil {
			return nil
		}
		b, e := d.CreateBufferChecked(bytes, wgpu.BufferUniform|wgpu.BufferCopyDst)
		if e != nil {
			err = fmt.Errorf("lights: a window's light pictures: %w", e)
		}
		return b
	}
	target := wgpu.UsageRenderAttachment | wgpu.UsageTextureBinding
	p = &lightPictures{
		occluders: tex(ow, oh, occluderFormat, target),
		light:     tex(lw, lh, lightFormat, target),
		mask:      tex(lw, lh, maskFormat, target),
		probes:    tex(draw.MaxProbes, 1, probeFormat, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc),
	}
	for i, f := range []wgpu.TextureFormat{shadowFormat, stretchFormat} {
		p.shadows[i] = tex(shadowAngles, draw.MaxShadowLights, f, target)
	}
	for i := range maxSuns {
		p.suns[i] = tex(texels(float32(ow)*sunScale), texels(float32(oh)*sunScale), sunFormat, target)
	}
	p.shadeMap, p.exposure = buffer(shadeMapBytes), buffer(16)
	if err != nil {
		for _, t := range made {
			t.Release()
		}
		for _, b := range []*wgpu.Buffer{p.shadeMap, p.exposure} {
			if b != nil {
				b.Release()
			}
		}
		return nil, err
	}
	r.bindLightPictures(p)
	l.pictures[win] = p
	return p, nil
}

// bindLightPictures makes the bind groups reading p's pictures.
func (r *webgpu2D) bindLightPictures(p *lightPictures) {
	l, d := &r.lights, r.dev
	for i := range p.shadows {
		p.shadowGroup[i] = d.CreateBindGroup(
			l.shadowSet,
			wgpu.BindEntry{Binding: 0, Buffer: l.rowsBuf, Size: l.rowsBuf.Size},
			wgpu.BindEntry{Binding: 1, View: p.occluders.View()},
		)
		p.pictureGroup[i] = d.CreateBindGroup(l.pictureSet, wgpu.BindEntry{Binding: 0, View: p.shadows[i].View()},
			wgpu.BindEntry{Binding: 1, Buffer: l.rowsBuf, Size: l.rowsBuf.Size},
			wgpu.BindEntry{Binding: 2, Buffer: l.lightsBuf, Size: l.lightsBuf.Size},
			wgpu.BindEntry{Binding: 3, Buffer: l.pointsBuf, Size: l.pointsBuf.Size})
	}
	for i := range maxSuns {
		p.sunShadowGroups[i] = d.CreateBindGroup(
			l.sunShadowSet,
			wgpu.BindEntry{Binding: 0, Buffer: l.sunParams[i], Size: 16},
			wgpu.BindEntry{Binding: 1, View: p.occluders.View()},
		)
		p.sunGroups[i] = d.CreateBindGroup(
			l.sunSet,
			wgpu.BindEntry{Binding: 0, View: p.suns[i].View()},
			wgpu.BindEntry{
				Binding: 1,
				Sampler: l.sampler,
			},
			wgpu.BindEntry{Binding: 2, Buffer: l.sunColors[i], Size: 32},
		)
	}
	p.shadeIndex = make([]float32, shadeMapBytes/4)
	p.read = r.readGroup(p)
	p.maskGroup = d.CreateBindGroup(l.maskSet, wgpu.BindEntry{Binding: 0, View: p.mask.View()})
	p.plain = d.CreateBindGroup(
		l.lightSet,
		wgpu.BindEntry{Binding: 0, View: p.light.View()},
		wgpu.BindEntry{Binding: 1, Sampler: l.sampler},
	)
	p.probeGroup = d.CreateBindGroup(
		l.probeSet,
		wgpu.BindEntry{Binding: 0, Buffer: l.probesBuf, Size: l.probesBuf.Size},
		wgpu.BindEntry{Binding: 1, View: p.light.View()},
		wgpu.BindEntry{Binding: 2, Sampler: l.sampler},
	)
}

// shadeMapBytes is a shade map's size: a float for each key a drawing
// can carry (draw.MaxKeys).
const shadeMapBytes = draw.MaxKeys * 4

// readGroup is the bind group the 2D shader reads p's light pictures by,
// the light picture standing for shaded ones not made yet.
func (r *webgpu2D) readGroup(p *lightPictures) *wgpu.BindGroup {
	l := &r.lights
	view := func(i int) wgpu.TextureView {
		if t := p.shaded[i]; t != nil {
			return t.View()
		}
		return p.light.View()
	}
	return r.dev.CreateBindGroup(
		l.readSet,
		wgpu.BindEntry{Binding: 0, View: p.light.View()},
		wgpu.BindEntry{Binding: 1, Sampler: l.sampler},
		wgpu.BindEntry{
			Binding: 2,
			View:    view(0),
		},
		wgpu.BindEntry{Binding: 3, View: view(1)},
		wgpu.BindEntry{Binding: 4, View: view(2)},
		wgpu.BindEntry{Binding: 5, Buffer: p.shadeMap, Size: shadeMapBytes},
		wgpu.BindEntry{Binding: 6, View: view(3)},
		wgpu.BindEntry{Binding: 7, Buffer: p.exposure, Size: 16},
	)
}

// shade makes p's shaded light picture i, and the groups drawing it, the
// first time it is needed.
func (r *webgpu2D) shade(p *lightPictures, i int) {
	if p.shaded[i] != nil {
		return
	}
	l, d := &r.lights, r.dev
	if p.shadeShadows[0] == nil {
		for s, f := range []wgpu.TextureFormat{shadowFormat, stretchFormat} {
			p.shadeShadows[s] = d.CreateTexture(
				shadowAngles,
				draw.MaxShadowLights,
				f,
				wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding,
			)
		}
	}
	p.shaded[i] = d.CreateTexture(
		p.light.Width,
		p.light.Height,
		lightFormat,
		wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding,
	)
	for s := range 2 {
		p.shadeShadowGroup[i][s] = d.CreateBindGroup(
			l.shadowSet,
			wgpu.BindEntry{Binding: 0, Buffer: l.shadeRows[i], Size: l.shadeRows[i].Size},
			wgpu.BindEntry{Binding: 1, View: p.occluders.View()},
		)
		p.shadePictureGroup[i][s] = d.CreateBindGroup(
			l.pictureSet,
			wgpu.BindEntry{Binding: 0, View: p.shadeShadows[s].View()},
			wgpu.BindEntry{Binding: 1, Buffer: l.shadeRows[i], Size: l.shadeRows[i].Size},
			wgpu.BindEntry{Binding: 2, Buffer: l.lightsBuf, Size: l.lightsBuf.Size},
			wgpu.BindEntry{Binding: 3, Buffer: l.pointsBuf, Size: l.pointsBuf.Size},
		)
	}
	for j := range maxSuns {
		p.shadeSunShadowGroups[i][j] = d.CreateBindGroup(
			l.sunShadowSet,
			wgpu.BindEntry{Binding: 0, Buffer: l.shadeSuns[i][j], Size: 16},
			wgpu.BindEntry{Binding: 1, View: p.occluders.View()},
		)
	}
	p.read.Release()
	p.read = r.readGroup(p)
}

func texels(v float32) int { return max(int(math.Ceil(float64(v))), 1) }

// lightFrame counts one window's lights laid out for the GPU: the shadow
// rows, the lights drawn as quads and their points, and the directional
// ones.
type lightFrame struct {
	rows, lights, points int
	suns                 []draw.Light
	// stretch is 1 when a light has an Elevation, a size or a fade: the
	// shadow maps then keep two stretches a texel and the light picture
	// reads them all; else 0, the plain shaders.
	stretch int
}

// layout fills l's rows, lights, points and vertices from li's lights, in
// the light rectangle rect's points.
func (l *lighting) layout(li *draw.Lighting, rect geom.Rect) lightFrame {
	var f lightFrame
	l.vertices, l.reach = l.vertices[:0], l.reach[:0]
	l.drawnAs = l.drawnAs[:0]
	origin := geom.V2(rect.X, rect.Y)
	for _, li := range li.Lights {
		l.drawnAs = append(l.drawnAs, -1)
		if li.Kind == draw.DirectionalLight {
			if len(f.suns) < maxSuns {
				l.drawnAs[len(l.drawnAs)-1] = -2 - len(f.suns)
				f.suns = append(f.suns, li)
			}
			continue
		}
		if f.lights >= len(l.lights) || len(li.Points) > len(l.points)-f.points {
			continue
		}
		at := li.At.Sub(origin)
		g := gpuLight{
			A:     [4]float32{at.X, at.Y, li.Radius, li.Falloff},
			Color: [4]float32{li.Color.R, li.Color.G, li.Color.B, float32(li.Kind)},
			Tint:  [4]float32{li.ShadowTint.R, li.ShadowTint.G, li.ShadowTint.B, 0},
		}
		bounds := geom.Rect{X: li.At.X - li.Radius, Y: li.At.Y - li.Radius, W: 2 * li.Radius, H: 2 * li.Radius}
		switch li.Kind {
		case draw.ConeLight:
			half := float64(min(li.Spread, 2*math.Pi)) / 2
			c := float32(math.Cos(half))
			feather := max(min(float32(math.Cos(half*0.85))-c, 0.2), 1e-3)
			g.Cone = [4]float32{li.Dir.X, li.Dir.Y, c, feather}
		case draw.ShapeLight:
			g.Cone = [4]float32{li.Dir.X, li.Dir.Y, 0, 0}
			g.Shape = [4]float32{float32(li.Shape), float32(f.points), float32(len(li.Points)), 0}
			g.Size = [4]float32{li.Size.X, li.Size.Y, 0, 0}
			for _, p := range li.Points {
				q := p.Sub(li.At)
				l.points[f.points] = [4]float32{q.X, q.Y, 0, 0}
				f.points++
			}
			b := li.ShapeBounds()
			bounds = geom.Rect{X: b.X - li.Radius, Y: b.Y - li.Radius, W: b.W + 2*li.Radius, H: b.H + 2*li.Radius}
		case draw.PointLight, draw.DirectionalLight:
		}
		if !li.XRay && !li.Unblocked {
			l.castRows(&f, &g, &li, bounds, origin)
		}
		l.lights[f.lights] = g
		l.drawnAs[len(l.drawnAs)-1] = f.lights
		reach := lightReach{bounds: bounds, groups: li.Groups, first: int(g.Shadow[0]), rows: int(g.Shadow[1])}
		reach.bounds.X, reach.bounds.Y = bounds.X-origin.X, bounds.Y-origin.Y
		reach.at = li.At.Sub(origin)
		l.reach = append(l.reach, reach)
		l.addQuad(li.At, bounds, origin, rect, float32(f.lights))
		f.lights++
	}
	return f
}

// castRows gives light li its shadow rows, one from each point it casts
// from, while the rows last, and sets g's shadow and stretch to read them.
func (l *lighting) castRows(f *lightFrame, g *gpuLight, li *draw.Light, bounds geom.Rect, origin geom.Vec2) {
	from, radius := []geom.Vec2{li.At}, li.Radius
	if li.Kind == draw.ShapeLight {
		from, radius = li.Samples, float32(math.Hypot(float64(bounds.W), float64(bounds.H)))
	}
	first := f.rows
	for _, s := range from {
		if f.rows >= len(l.rows) {
			break
		}
		rel := s.Sub(origin)
		elevation := float32(math.Round(float64(li.Elevation * draw.OccluderScale)))
		l.rows[f.rows] = [4]float32{
			rel.X * draw.OccluderScale, rel.Y * draw.OccluderScale,
			radius * draw.OccluderScale, float32(li.Groups) + 8*elevation,
		}
		f.rows++
	}
	if n := f.rows - first; n > 0 {
		g.Shadow = [4]float32{float32(first), float32(n), 0, 1.5 / max(radius*draw.OccluderScale, 1)}
		stretches := float32(0)
		if li.Elevation > 0 || li.ShadowFade > 0 {
			stretches, f.stretch = 1, 1
		}
		if li.ShadowSize > 0 {
			f.stretch = 1
		}
		g.Height = [4]float32{
			li.ShadowSize * draw.OccluderScale,
			float32(li.ShadowFade),
			stretches,
			li.FadeFrom,
		}
	}
}

// addQuad adds the two triangles covering bounds for the light at at, the
// index'th, in rect's clip space.
func (l *lighting) addQuad(at geom.Vec2, bounds geom.Rect, origin geom.Vec2, rect geom.Rect, index float32) {
	corner := func(x, y float32) lightVertex {
		q := geom.V2(x, y).Sub(origin)
		local := geom.V2(x, y).Sub(at)
		return lightVertex{
			NDC:   [2]float32{q.X/rect.W*2 - 1, 1 - q.Y/rect.H*2},
			Local: [2]float32{local.X, local.Y},
			Index: index,
		}
	}
	a, b := corner(bounds.X, bounds.Y), corner(bounds.X+bounds.W, bounds.Y)
	c, d := corner(bounds.X+bounds.W, bounds.Y+bounds.H), corner(bounds.X, bounds.Y+bounds.H)
	l.vertices = append(l.vertices, a, b, c, a, c, d)
}

// lightSpans are where a window's lighting lists lie in its buffers, nil
// for those with nothing to draw.
type lightSpans struct {
	occluders, emissive, ambient *span
	// silhouettes are the spans of the Lighting's Silhouettes' lists.
	silhouettes []*span
}

// light draws win's lighting before anything of win reads it: what blocks
// light, each shadow-casting light's rows of the shadow maps and each
// directional light's shadow picture, then the light picture, the ambient
// plus each light. Without lighting, win's lit sprites read white.
func (r *webgpu2D) light(
	e *wgpu.Encoder,
	win *platform.Window,
	li *draw.Lighting,
	spans lightSpans,
	fb geom.Vec2,
	timers *wgpu.Timers,
) {
	occluders, emissive := spans.occluders, spans.emissive
	l := &r.lights
	l.current, l.shading = l.unlit, false
	r.answer(win)
	if li == nil {
		return
	}
	size := win.Size()
	rect := draw.LightRect(size)
	scale := fb.X / max(size.X, 1)
	p, err := r.lightPicturesFor(win, geom.V2(rect.W, rect.H), geom.V2(rect.W*scale, rect.H*scale))
	if err != nil {
		panic(fmt.Errorf("gpu: %w", err))
	}

	r.makeGlows(e, li)
	r.drawOccluders(e, p, occluders, rect, timers)

	f := l.layout(li, rect)
	if l.lastRows == nil {
		l.lastRows, l.lastStretch = map[*platform.Window]int{}, map[*platform.Window]int{}
	}
	l.lastRows[win], l.lastStretch[win] = f.rows, f.stretch
	r.uploadLights(e, p, li, f, rect, timers)
	picture := func(target *wgpu.Texture, group *wgpu.BindGroup, span string) {
		amb := li.Ambient
		into := geom.V2(float32(target.Width), float32(target.Height))
		pp := e.BeginPass(target.View(), &wgpu.Color{R: float64(amb.R), G: float64(amb.G), B: float64(amb.B), A: 1},
			timers.Span(span))
		if spans.ambient != nil {
			r.record(
				pp,
				*spans.ambient,
				geom.V2(rect.W, rect.H),
				into,
				target2D{format: lightFormat, linear: true},
				nil,
			)
		}
		if n := len(l.vertices); n > 0 {
			pp.SetPipeline(l.picturePipe[f.stretch])
			pp.SetBindGroup(0, group)
			pp.SetVertexBuffer(l.lightVerts, 0, n*int(unsafe.Sizeof(lightVertex{})))
			pp.Draw(n, 0)
		}
		r.drawStatic(pp, li.Static, rect)
		for i := range f.suns {
			pp.SetPipeline(l.sunPipe)
			pp.SetBindGroup(0, p.sunGroups[i])
			pp.Draw(3, 0)
		}
		if emissive != nil {
			r.record(pp, *emissive, geom.V2(rect.W, rect.H), into, target2D{format: lightFormat, linear: true}, nil)
		}
		pp.End()
	}
	picture(p.light, p.pictureGroup[f.stretch], "light picture")
	all := func(int) bool { return true }
	r.silhouettes(
		e,
		p,
		li,
		spans,
		rect,
		f,
		p.light,
		all,
		func(uint8) bool { return true },
		false,
		func(int) *wgpu.BindGroup { return p.pictureGroup[f.stretch] },
		timers,
	)
	r.probe(e, win, li, p, rect, timers)
	r.shaded(e, p, li, spans, rect, f, picture, timers)
	r.writeExposure(p, li)
	l.current, l.shading = p.read, len(li.Plan.Pictures) > 0
}

// writeExposure writes li's exposure, 1 when unset, for p's readers.
func (r *webgpu2D) writeExposure(p *lightPictures, li *draw.Lighting) {
	l := &r.lights
	if l.exposed == nil {
		l.exposed = make([]float32, 4)
	}
	l.exposed[0] = 1
	if li.Exposure > 0 {
		l.exposed[0] = li.Exposure
	}
	r.dev.WriteBuffer(p.exposure, 0, unsafe.Pointer(&l.exposed[0]), 16)
}

// drawOccluders draws what blocks light into p's occluder map.
func (r *webgpu2D) drawOccluders(
	e *wgpu.Encoder,
	p *lightPictures,
	occluders *span,
	rect geom.Rect,
	timers *wgpu.Timers,
) {
	pass := e.BeginPass(p.occluders.View(), &wgpu.Color{}, timers.Span("light occluders"))
	if occluders != nil {
		r.record(
			pass,
			*occluders,
			geom.V2(rect.W, rect.H),
			geom.V2(float32(p.occluders.Width), float32(p.occluders.Height)),
			target2D{format: occluderFormat},
			nil,
		)
	}
	pass.End()
}

// uploadLights writes the frame's shadow rows, lights, points, suns and
// light quads to their buffers, casting the shadow maps and each sun's
// shadow picture on the way.
func (r *webgpu2D) uploadLights(
	e *wgpu.Encoder,
	p *lightPictures,
	li *draw.Lighting,
	f lightFrame,
	rect geom.Rect,
	timers *wgpu.Timers,
) {
	l := &r.lights
	if f.rows > 0 {
		r.dev.WriteBuffer(l.rowsBuf, 0, unsafe.Pointer(&l.rows[0]), f.rows*16)
		pass := e.BeginPass(
			p.shadows[f.stretch].View(),
			&wgpu.Color{R: noShadow, G: noShadow, B: noShadow, A: noShadow},
			timers.Span("light shadows"),
		)
		pass.SetPipeline(l.shadowPipe[f.stretch])
		pass.SetBindGroup(0, p.shadowGroup[f.stretch])
		pass.SetScissor(0, 0, shadowAngles, f.rows)
		pass.Draw(3, 0)
		pass.End()
	}
	if f.lights > 0 {
		r.packSilhouettes(p, li, rect)
		r.dev.WriteBuffer(l.lightsBuf, 0, unsafe.Pointer(&l.lights[0]), f.lights*int(unsafe.Sizeof(gpuLight{})))
	}
	if f.points > 0 {
		r.dev.WriteBuffer(l.pointsBuf, 0, unsafe.Pointer(&l.points[0]), f.points*16)
	}
	for i, sun := range f.suns {
		length := sun.Length * draw.OccluderScale
		if sun.XRay {
			length = 0
		}
		l.suns[3*i] = [4]float32{sun.Dir.X, sun.Dir.Y, length, float32(sun.Groups)}
		l.suns[3*i+1] = [4]float32{sun.Color.R, sun.Color.G, sun.Color.B, 1}
		l.suns[3*i+2] = [4]float32{sun.ShadowTint.R, sun.ShadowTint.G, sun.ShadowTint.B, 0}
		r.dev.WriteBuffer(l.sunParams[i], 0, unsafe.Pointer(&l.suns[3*i]), 16)
		r.dev.WriteBuffer(l.sunColors[i], 0, unsafe.Pointer(&l.suns[3*i+1]), 32)
		r.sunShadow(e, p.suns[i], p.sunShadowGroups[i], timers)
	}
	if n := len(l.vertices); n > 0 {
		bytes := n * int(unsafe.Sizeof(lightVertex{}))
		l.lightVerts = r.grow(l.lightVerts, bytes, wgpu.BufferVertex|wgpu.BufferCopyDst)
		r.dev.WriteBuffer(l.lightVerts, 0, unsafe.Pointer(&l.vertices[0]), bytes)
	}
}

// silhouettes takes out of target, a light picture of win's, the light
// its Silhouettes shadow: those of the sets pick chooses, of the quads
// whose groups quads keeps; with back, it adds that light back instead,
// to undo a picture's copy of them. On each
// mask atlas page, the page's silhouettes are drawn into the page,
// projected on the GPU, an instance a quad, then each light whose tile is
// there and chosen has its part, times its tile, taken out or added,
// drawn with the bind group groupOf gives for its quad. A directional
// light's are drawn into the mask, then its part taken out or added.
func (r *webgpu2D) silhouettes(
	e *wgpu.Encoder,
	p *lightPictures,
	li *draw.Lighting,
	spans lightSpans,
	rect geom.Rect,
	f lightFrame,
	target *wgpu.Texture,
	pick func(set int) bool,
	quads func(groups uint8) bool,
	back bool,
	groupOf func(int) *wgpu.BindGroup,
	timers *wgpu.Timers,
) {
	l := &r.lights
	vertex := int(unsafe.Sizeof(lightVertex{}))
	instance := int(unsafe.Sizeof(silInstance{}))
	pipe, sunPipe := l.silhouettePipe[f.stretch], l.sunSilhouettePipe
	if back {
		pipe, sunPipe = l.silhouetteBackPipe[f.stretch], l.sunSilhouetteBackPipe
	}
	for page := range p.atlas {
		l.drawn = l.drawn[:0]
		for _, c := range l.casting {
			if c.page == page && pick(c.set) {
				l.drawn = append(l.drawn, c)
			}
		}
		if len(l.drawn) == 0 {
			continue
		}
		pass := e.BeginPass(p.atlas[page].View(), &wgpu.Color{}, timers.Span("light silhouettes"))
		pass.SetPipeline(l.silMaskPipe)
		pass.SetBindGroup(0, l.castGroup)
		for _, b := range l.buckets {
			if b.page == page && quads(b.groups) {
				pass.SetBindGroup(1, r.pictureGroup(b.texture))
				pass.SetVertexBuffer(l.instBuf, b.first*instance, b.count*instance)
				pass.DrawInstanced(6, b.count)
			}
		}
		pass.End()
		pass = e.BeginPass(target.View(), nil, timers.Span("light silhouettes"))
		pass.SetPipeline(pipe)
		pass.SetBindGroup(1, p.atlasGroups[page])
		for _, c := range l.drawn {
			pass.SetBindGroup(0, groupOf(c.k))
			pass.SetVertexBuffer(l.silVertBuf, c.vertex*vertex, 6*vertex)
			pass.Draw(6, 0)
		}
		pass.End()
	}
	size := geom.V2(float32(p.mask.Width), float32(p.mask.Height))
	for n, s := range li.Silhouettes {
		if s.List == nil || !pick(n) || !quads(s.Groups) || s.Light >= len(l.drawnAs) || l.drawnAs[s.Light] > -2 ||
			spans.silhouettes[n] == nil {
			continue
		}
		pass := e.BeginPass(p.mask.View(), &wgpu.Color{}, timers.Span("light silhouettes"))
		r.record(pass, *spans.silhouettes[n], geom.V2(rect.W, rect.H), size, target2D{format: maskFormat}, nil)
		pass.End()
		pass = e.BeginPass(target.View(), nil, timers.Span("light silhouettes"))
		pass.SetPipeline(sunPipe)
		pass.SetBindGroup(0, p.sunGroups[-2-l.drawnAs[s.Light]])
		pass.SetBindGroup(1, p.maskGroup)
		pass.Draw(3, 0)
		pass.End()
	}
}

// silInstance is a silhouette quad as the mask shader reads it: where its
// sprite stands, how wide a point of it and how tall it is, its rectangle
// in its own space, its picture's uv corners and the light casting it.
type silInstance struct {
	Foot  [2]float32
	Size  [2]float32
	Rect  [4]float32
	UV    [4]float32
	Light float32
	_     [3]float32
}

// silKey tells silhouette buckets apart.
type silKey struct {
	page    int
	groups  uint8
	texture *render.Texture
}

// silBucket is a run of the frame's silhouette instances drawn alike: on
// one atlas page, of one set of groups, from one texture.
type silBucket struct {
	page         int
	groups       uint8
	texture      *render.Texture
	first, count int
}

// pictureGroup is the bind group the silhouette mask shader reads t by.
func (r *webgpu2D) pictureGroup(t *render.Texture) *wgpu.BindGroup {
	l := &r.lights
	view := r.image(t).View()
	if g, ok := l.pictureGroups[view]; ok {
		return g
	}
	g := r.dev.CreateBindGroup(
		l.pictureSetLayout,
		wgpu.BindEntry{Binding: 0, View: view},
		wgpu.BindEntry{Binding: 1, Sampler: r.sampler(t)},
	)
	l.pictureGroups[view] = g
	return g
}

// packSilhouettes readies the lights drawn as quads that cast
// silhouettes: their quads over where their silhouettes can fall within
// their reach, and, in their uniforms, how far their tile of the mask
// atlas stands from where they shine, in pixels, which their silhouette
// shader adds to read it; the silhouette instances, in runs a page, set of
// groups and texture; and the pages the atlas needs.
func (r *webgpu2D) packSilhouettes(p *lightPictures, li *draw.Lighting, rect geom.Rect) {
	l := &r.lights
	l.casting, l.silVerts, l.buckets = l.casting[:0], l.silVerts[:0], l.buckets[:0]
	if li.SilhouetteSide <= 0 {
		return
	}
	kpx := float32(p.light.Width) / rect.W
	pages := 0
	l.castOf = slices.Grow(l.castOf[:0], len(li.Silhouettes))[:len(li.Silhouettes)]
	for n, s := range li.Silhouettes {
		l.castOf[n] = -1
		if s.List != nil || s.Light >= len(l.drawnAs) || l.drawnAs[s.Light] < 0 {
			continue
		}
		k := l.drawnAs[s.Light]
		b := l.reach[k].bounds
		x0, y0 := max(b.X, s.Area.X), max(b.Y, s.Area.Y)
		x1, y1 := min(b.X+b.W, s.Area.X+s.Area.W), min(b.Y+b.H, s.Area.Y+s.Area.H)
		if x1 <= x0 || y1 <= y0 {
			continue
		}
		corner := func(x, y float32) lightVertex {
			q := geom.V2(x, y)
			local := q.Sub(l.reach[k].at)
			return lightVertex{
				NDC:   [2]float32{q.X/rect.W*2 - 1, 1 - q.Y/rect.H*2},
				Local: [2]float32{local.X, local.Y},
				Index: float32(k),
			}
		}
		a, bb, c, d := corner(x0, y0), corner(x1, y0), corner(x1, y1), corner(x0, y1)
		l.castOf[n] = k
		l.casting = append(l.casting, casting{set: n, k: k, page: s.Page, vertex: len(l.silVerts)})
		l.silVerts = append(l.silVerts, a, bb, c, a, c, d)
		shift := geom.V2(s.Tile.X-s.Area.X, s.Tile.Y-s.Area.Y)
		l.lights[k].Size[2], l.lights[k].Size[3] = shift.X*kpx, shift.Y*kpx
		l.casts[2*k] = [4]float32{shift.X, shift.Y, li.Lights[s.Light].Elevation, 0}
		l.casts[2*k+1] = [4]float32{s.Area.X, s.Area.Y, s.Area.W, s.Area.H}
		pages = max(pages, s.Page+1)
	}
	if n := len(l.silVerts); n > 0 {
		bytes := n * int(unsafe.Sizeof(lightVertex{}))
		l.silVertBuf = r.grow(l.silVertBuf, bytes, wgpu.BufferVertex|wgpu.BufferCopyDst)
		r.dev.WriteBuffer(l.silVertBuf, 0, unsafe.Pointer(&l.silVerts[0]), bytes)
		r.dev.WriteBuffer(l.castsBuf, 0, unsafe.Pointer(&l.casts[0]), len(l.casts)*16)
		l.pageSide[0] = li.SilhouetteSide
		r.dev.WriteBuffer(l.pageBuf, 0, unsafe.Pointer(&l.pageSide[0]), 16)
	}
	r.instances(li)
	side := int(math.Ceil(float64(li.SilhouetteSide * kpx)))
	for pages > len(p.atlas) || len(p.atlas) > 0 && p.atlas[0].Width != side {
		r.atlasPage(p, side)
	}
}

// instances lays the frame's silhouette quads out as instances, in runs
// of one page, set of groups and texture, and writes them to the GPU.
func (r *webgpu2D) instances(li *draw.Lighting) {
	l := &r.lights
	l.order = l.order[:0]
	for i, q := range li.SilhouetteQuads {
		if l.castOf[q.Set] >= 0 {
			l.order = append(l.order, int32(i))
		}
	}
	if len(l.order) == 0 {
		return
	}
	clear(l.bucketOf)
	l.bucketOfQuad = slices.Grow(l.bucketOfQuad[:0], len(l.order))[:len(l.order)]
	for j, i := range l.order {
		q := &li.SilhouetteQuads[i]
		k := silKey{li.Silhouettes[q.Set].Page, q.Groups, q.Texture}
		b, ok := l.bucketOf[k]
		if !ok {
			b = len(l.buckets)
			l.bucketOf[k] = b
			l.buckets = append(l.buckets, silBucket{page: k.page, groups: k.groups, texture: k.texture})
		}
		l.buckets[b].count++
		l.bucketOfQuad[j] = b
	}
	first := 0
	for b := range l.buckets {
		l.buckets[b].first = first
		first += l.buckets[b].count
		l.buckets[b].count = 0
	}
	l.instanceData = slices.Grow(l.instanceData[:0], len(l.order))[:len(l.order)]
	for j, i := range l.order {
		q := &li.SilhouetteQuads[i]
		b := &l.buckets[l.bucketOfQuad[j]]
		l.instanceData[b.first+b.count] = silInstance{
			Foot: [2]float32{q.Foot.X, q.Foot.Y}, Size: [2]float32{q.Across, q.Height},
			Rect: [4]float32{q.Rect.X, q.Rect.Y, q.Rect.W, q.Rect.H}, UV: q.UV, Light: float32(l.castOf[q.Set]),
		}
		b.count++
	}
	bytes := len(l.instanceData) * int(unsafe.Sizeof(silInstance{}))
	l.instBuf = r.grow(l.instBuf, bytes, wgpu.BufferVertex|wgpu.BufferCopyDst)
	r.dev.WriteBuffer(l.instBuf, 0, unsafe.Pointer(&l.instanceData[0]), bytes)
}

// atlasPage adds a page to p's mask atlas, side pixels square, dropping
// the pages there are when they are of another side.
func (r *webgpu2D) atlasPage(p *lightPictures, side int) {
	if len(p.atlas) > 0 && p.atlas[0].Width != side {
		for i := range p.atlas {
			p.atlas[i].Release()
			p.atlasGroups[i].Release()
		}
		p.atlas, p.atlasGroups = nil, nil
	}
	t := r.dev.CreateTexture(side, side, maskFormat, wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding)
	p.atlas = append(p.atlas, t)
	p.atlasGroups = append(
		p.atlasGroups,
		r.dev.CreateBindGroup(r.lights.maskSet, wgpu.BindEntry{Binding: 0, View: t.View()}),
	)
}

// casting is a light drawn as a quad that casts silhouettes: its set in
// the Lighting's Silhouettes, its index among the lights drawn, its mask
// atlas page and where its quad starts in the silhouette vertices.
type casting struct {
	set, k, page int
	vertex       int
}

// sunShadow draws a directional light's shadow picture into target, its
// uniforms in group.
func (r *webgpu2D) sunShadow(e *wgpu.Encoder, target *wgpu.Texture, group *wgpu.BindGroup, timers *wgpu.Timers) {
	pass := e.BeginPass(target.View(), &wgpu.Color{R: 1}, timers.Span("light sun"))
	pass.SetPipeline(r.lights.sunShadowPipe)
	pass.SetBindGroup(0, group)
	pass.Draw(3, 0)
	pass.End()
}

// shaded draws p's other light pictures, as li's Plan says, up to
// maxShades, and fills p's shade map. Each is the light picture without
// the lights the Plan leaves out, and with only the lights an occluder of
// the groups it leaves out can shade redrawn: those taken out as drawn
// and added back with shadow maps cast by the groups kept, the lights
// left out only taken out, their silhouettes added back. With directional
// lights and occluders left out, which cross the whole view, it is drawn
// whole again as picture draws it, then the lights left out taken out;
// those are drawn last, as they redraw the directional lights' shadows.
func (r *webgpu2D) shaded(
	e *wgpu.Encoder,
	p *lightPictures,
	li *draw.Lighting,
	spans lightSpans,
	rect geom.Rect,
	f lightFrame,
	picture func(*wgpu.Texture, *wgpu.BindGroup, string),
	timers *wgpu.Timers,
) {
	l := &r.lights
	clear(p.shadeIndex)
	plan := &li.Plan
	if bytes := 2 * len(l.vertices) * maxShades * int(unsafe.Sizeof(lightVertex{})); bytes > 0 {
		l.shadeVertBuf = r.grow(l.shadeVertBuf, bytes, wgpu.BufferVertex|wgpu.BufferCopyDst)
	}
	off := 0
	for _, last := range []bool{false, true} {
		for i, pic := range plan.Pictures {
			if i >= maxShades {
				break
			}
			keepGroups := uint8(draw.AllGroups) &^ pic.Left
			keep := int(keepGroups)
			l.excluded = l.excluded[:0]
			if pic.Left != 0 {
				l.excluded = excludedBy(li.Occluders, pic.Left, l.excluded)
			}
			whole := len(f.suns) > 0 && len(l.excluded) > 0
			if whole != last {
				continue
			}
			r.shade(p, i)
			l.outOf(&pic)
			r.writeShadeRows(i, f, keep)
			l.reached = l.reached[:0]
			if !whole && pic.Left != 0 {
				l.reached = reached(l.reach, l.excluded, pic.Left, l.reached)
				l.reached = slices.DeleteFunc(l.reached, func(k int) bool { return slices.Contains(l.out, k) })
			}
			r.shadeShadows(e, p, i, f, whole, timers)
			if whole {
				off = r.shadeWhole(e, p, li, spans, rect, f, i, &pic, keepGroups, off, picture, timers)
			} else {
				off = r.shadeRedrawn(e, p, li, spans, rect, f, i, &pic, keepGroups, off, timers)
			}
			for key, of := range plan.Of {
				if int(of) == i+1 {
					p.shadeIndex[key] = float32(i + 1)
				}
			}
		}
	}
	r.dev.WriteBuffer(p.shadeMap, 0, unsafe.Pointer(&p.shadeIndex[0]), shadeMapBytes)
}

// outOf fills l.out and l.outSuns with the lights pic leaves out: the
// lights by their index, the suns by theirs.
func (l *lighting) outOf(pic *draw.Picture) {
	l.out, l.outSuns = l.out[:0], l.outSuns[:0]
	for _, n := range pic.Excluded {
		switch k := l.drawnAs[n]; {
		case k >= 0:
			l.out = append(l.out, k)
		case k <= -2:
			l.outSuns = append(l.outSuns, -2-k)
		}
	}
}

// writeShadeRows writes the frame's shadow rows to light picture i's, each
// cast only by its groups in keep.
func (r *webgpu2D) writeShadeRows(i int, f lightFrame, keep int) {
	l := &r.lights
	rows := l.shadeData[:f.rows]
	for j, row := range l.rows[:f.rows] {
		groups := int(row[3]) % 8
		row[3] += float32(groups&keep - groups)
		rows[j] = row
	}
	if f.rows > 0 {
		r.dev.WriteBuffer(l.shadeRows[i], 0, unsafe.Pointer(&rows[0]), f.rows*16)
	}
}

// shadeWhole draws light picture i whole, its suns' shadows and every
// light's cast only by the groups kept, then takes out the lights pic
// leaves out; it gives the offset of the next vertices in the shade buffer.
func (r *webgpu2D) shadeWhole(
	e *wgpu.Encoder,
	p *lightPictures,
	li *draw.Lighting,
	spans lightSpans,
	rect geom.Rect,
	f lightFrame,
	i int,
	pic *draw.Picture,
	keepGroups uint8,
	off int,
	picture func(*wgpu.Texture, *wgpu.BindGroup, string),
	timers *wgpu.Timers,
) int {
	l := &r.lights
	keep := int(keepGroups)
	shaded := p.shadePictureGroup[i][f.stretch]
	out := func(set int) bool { return slices.Contains(pic.Excluded, li.Silhouettes[set].Light) }
	kept := func(g uint8) bool { return g&keepGroups != 0 }
	for j := range f.suns {
		sun := &l.shadeData[len(l.rows)+i*maxSuns+j]
		*sun = l.suns[3*j]
		sun[3] = float32(int(sun[3]) & keep)
		r.dev.WriteBuffer(l.shadeSuns[i][j], 0, unsafe.Pointer(sun), 16)
		r.sunShadow(e, p.suns[j], p.shadeSunShadowGroups[i][j], timers)
	}
	picture(p.shaded[i], shaded, "light shade")
	l.redrawn = -1
	off = r.takeOut(e, p, i, f, off, shaded, timers)
	r.silhouettes(e, p, li, spans, rect, f, p.shaded[i], func(n int) bool { return !out(n) }, kept, false,
		func(int) *wgpu.BindGroup { return shaded }, timers)
	return off
}

// shadeRedrawn draws light picture i as the window's, redrawing only the
// lights the groups left out reach; it gives the offset of the next
// vertices in the shade buffer.
func (r *webgpu2D) shadeRedrawn(
	e *wgpu.Encoder,
	p *lightPictures,
	li *draw.Lighting,
	spans lightSpans,
	rect geom.Rect,
	f lightFrame,
	i int,
	pic *draw.Picture,
	keepGroups uint8,
	off int,
	timers *wgpu.Timers,
) int {
	l := &r.lights
	shaded, base := p.shadePictureGroup[i][f.stretch], p.pictureGroup[f.stretch]
	out := func(set int) bool { return slices.Contains(pic.Excluded, li.Silhouettes[set].Light) }
	kept := func(g uint8) bool { return g&keepGroups != 0 }
	every := func(uint8) bool { return true }
	off = r.redraw(e, p, i, f, off, timers)
	l.redrawn = len(l.reached)
	redrawn := func(n int) bool { return slices.Contains(l.reached, l.castOf[n]) }
	r.silhouettes(
		e,
		p,
		li,
		spans,
		rect,
		f,
		p.shaded[i],
		func(n int) bool { return out(n) || redrawn(n) },
		every,
		true,
		func(int) *wgpu.BindGroup { return base },
		timers,
	)
	if pic.Left != 0 {
		r.silhouettes(
			e,
			p,
			li,
			spans,
			rect,
			f,
			p.shaded[i],
			func(n int) bool { return !out(n) && !redrawn(n) },
			func(g uint8) bool { return !kept(g) },
			true,
			func(int) *wgpu.BindGroup { return base },
			timers,
		)
		r.silhouettes(
			e,
			p,
			li,
			spans,
			rect,
			f,
			p.shaded[i],
			redrawn,
			kept,
			false,
			func(int) *wgpu.BindGroup { return shaded },
			timers,
		)
	}
	return off
}

// shadeShadows draws shaded picture i's shadow maps, cast by the groups
// kept: every light's rows when whole, else those of the lights reached.
func (r *webgpu2D) shadeShadows(
	e *wgpu.Encoder,
	p *lightPictures,
	i int,
	f lightFrame,
	whole bool,
	timers *wgpu.Timers,
) {
	l := &r.lights
	if f.rows == 0 || !whole && len(l.reached) == 0 {
		return
	}
	pass := e.BeginPass(
		p.shadeShadows[f.stretch].View(),
		&wgpu.Color{R: noShadow, G: noShadow, B: noShadow, A: noShadow},
		timers.Span("light shade shadows"),
	)
	pass.SetPipeline(l.shadowPipe[f.stretch])
	pass.SetBindGroup(0, p.shadeShadowGroup[i][f.stretch])
	if whole {
		pass.SetScissor(0, 0, shadowAngles, f.rows)
		pass.Draw(3, 0)
	}
	for _, k := range l.reached {
		if n := l.reach[k].rows; !whole && n > 0 {
			pass.SetScissor(0, l.reach[k].first, shadowAngles, n)
			pass.Draw(3, 0)
		}
	}
	pass.End()
}

// redraw draws shaded picture i as the light picture with the lights
// reached taken out and added back as the groups kept shade them, and
// those left out (out, outSuns) only taken out, their quads written at off
// in the shaded vertex buffer; it returns where the next picture's quads
// go.
func (r *webgpu2D) redraw(e *wgpu.Encoder, p *lightPictures, i int, f lightFrame, off int, timers *wgpu.Timers) int {
	l := &r.lights
	l.shadeVerts = l.shadeVerts[:0]
	for _, k := range l.reached {
		l.shadeVerts = append(l.shadeVerts, l.vertices[6*k:6*k+6]...)
	}
	added := len(l.shadeVerts)
	for _, k := range l.out {
		l.shadeVerts = append(l.shadeVerts, l.vertices[6*k:6*k+6]...)
	}
	pass := e.BeginPass(p.shaded[i].View(), nil, timers.Span("light shade"))
	pass.SetPipeline(l.copyPipe)
	pass.SetBindGroup(0, p.plain)
	pass.Draw(3, 0)
	if n := len(l.shadeVerts); n > 0 {
		bytes := n * int(unsafe.Sizeof(lightVertex{}))
		r.dev.WriteBuffer(l.shadeVertBuf, off, unsafe.Pointer(&l.shadeVerts[0]), bytes)
		pass.SetVertexBuffer(l.shadeVertBuf, off, bytes)
		pass.SetPipeline(l.subtractPipe[f.stretch])
		pass.SetBindGroup(0, p.pictureGroup[f.stretch])
		pass.Draw(n, 0)
		if added > 0 {
			pass.SetPipeline(l.picturePipe[f.stretch])
			pass.SetBindGroup(0, p.shadePictureGroup[i][f.stretch])
			pass.Draw(added, 0)
		}
		off += bytes
	}
	r.sunsOut(pass, p)
	pass.End()
	return off
}

// takeOut takes the lights left out (out, outSuns) out of shaded picture
// i, drawn whole with group, their quads written at off in the shaded
// vertex buffer; it returns where the next picture's quads go.
func (r *webgpu2D) takeOut(
	e *wgpu.Encoder,
	p *lightPictures,
	i int,
	f lightFrame,
	off int,
	group *wgpu.BindGroup,
	timers *wgpu.Timers,
) int {
	l := &r.lights
	if len(l.out) == 0 && len(l.outSuns) == 0 {
		return off
	}
	l.shadeVerts = l.shadeVerts[:0]
	for _, k := range l.out {
		l.shadeVerts = append(l.shadeVerts, l.vertices[6*k:6*k+6]...)
	}
	pass := e.BeginPass(p.shaded[i].View(), nil, timers.Span("light shade"))
	if n := len(l.shadeVerts); n > 0 {
		bytes := n * int(unsafe.Sizeof(lightVertex{}))
		r.dev.WriteBuffer(l.shadeVertBuf, off, unsafe.Pointer(&l.shadeVerts[0]), bytes)
		pass.SetVertexBuffer(l.shadeVertBuf, off, bytes)
		pass.SetPipeline(l.subtractPipe[f.stretch])
		pass.SetBindGroup(0, group)
		pass.Draw(n, 0)
		off += bytes
	}
	r.sunsOut(pass, p)
	pass.End()
	return off
}

// sunsOut takes the directional lights left out (outSuns) out of what
// pass draws into, one of p's pictures, as their shadow pictures stand.
func (r *webgpu2D) sunsOut(pass *wgpu.Pass, p *lightPictures) {
	l := &r.lights
	for _, j := range l.outSuns {
		pass.SetPipeline(l.sunSubtractPipe)
		pass.SetBindGroup(0, p.sunGroups[j])
		pass.Draw(3, 0)
	}
}

// lightReach is where a light of the frame reaches, in the light
// picture's points, the occluder groups that cast its shadows and its
// rows of the shadow maps.
type lightReach struct {
	bounds      geom.Rect
	at          geom.Vec2
	groups      uint8
	first, rows int
}

// excludedBy appends to out the rectangles round the occluders of occ, in
// the light picture's points, of any of the groups left: what a drawing
// ShadedBy the others is not shaded by.
func excludedBy(occ *draw.List, left uint8, out []geom.Rect) []geom.Rect {
	if occ == nil {
		return out
	}
	vs := occ.Vertices
	for t := 0; t+2 < len(occ.Indices); t += 3 {
		a, b, c := vs[occ.Indices[t]], vs[occ.Indices[t+1]], vs[occ.Indices[t+2]]
		if groupsOf(a.Color)&left == 0 {
			continue
		}
		lo := geom.V2(min(a.Pos.X, b.Pos.X, c.Pos.X), min(a.Pos.Y, b.Pos.Y, c.Pos.Y))
		hi := geom.V2(max(a.Pos.X, b.Pos.X, c.Pos.X), max(a.Pos.Y, b.Pos.Y, c.Pos.Y))
		out = append(out, geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y})
	}
	return out
}

// groupsOf reads the occluder groups an occluder's colour says (see
// draw.GroupColor).
func groupsOf(c color.Color) uint8 {
	var g uint8
	for i, v := range []float32{c.R, c.G, c.B} {
		if v > 0.5 {
			g |= 1 << i
		}
	}
	return g
}

// reached appends to out the lights, by index, an occluder in excluded
// can shade: one blocked by a group left, casting shadows, with one of
// them in its reach.
func reached(lights []lightReach, excluded []geom.Rect, left uint8, out []int) []int {
	for k, l := range lights {
		if l.rows == 0 || l.groups&left == 0 {
			continue
		}
		for _, o := range excluded {
			if o.X < l.bounds.X+l.bounds.W && l.bounds.X < o.X+o.W && o.Y < l.bounds.Y+l.bounds.H &&
				l.bounds.Y < o.Y+o.H {
				out = append(out, k)
				break
			}
		}
	}
	return out
}

// lightDebug draws, in pass over win's target of format and fb pixels,
// the lighting texture w's debug view asks for, if any and win is lit.
func (r *webgpu2D) lightDebug(
	w *world.World,
	pass *wgpu.Pass,
	win *platform.Window,
	format wgpu.TextureFormat,
	fb geom.Vec2,
) {
	dbg := draw.LightDebugOf(w, win)
	view := dbg.View
	l := &r.lights
	p := l.pictures[win]
	if view == draw.LightViewOff || p == nil {
		return
	}
	pipe := l.debugPipes[format]
	if pipe == nil {
		sh, err := r.dev.CreateShader("light debug", lightDebugWGSL)
		if err != nil {
			return
		}
		over := wgpu.Blend{
			Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.OneMinusSrcAlpha},
			Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.One},
		}
		pipe, err = r.dev.CreateRenderPipeline(
			wgpu.PipelineSpec{Label: "light debug", Shader: sh, Layouts: []*wgpu.BindGroupLayout{l.debugSet},
				Vertex: fullVS, Fragment: "debug_fs", Format: format, Blend: &over},
		)
		if err != nil {
			return
		}
		if l.debugPipes == nil {
			l.debugPipes = make(map[wgpu.TextureFormat]*wgpu.RenderPipeline)
		}
		l.debugPipes[format] = pipe
	}
	if l.debugBuf == nil {
		l.debugBuf = r.dev.CreateBuffer(64, wgpu.BufferUniform|wgpu.BufferCopyDst)
	}
	tex := p.light
	switch view {
	case draw.LightViewOccluders:
		tex = p.occluders
	case draw.LightViewShadows:
		tex = p.shadows[l.lastStretch[win]]
	case draw.LightViewOff, draw.LightViewPicture:
	}
	size := win.Size()
	rect := draw.LightRect(size)
	scale := fb.X / max(size.X, 1)
	panel, area := geom.Rect{}, dbg.Area
	if win == wm.Primary(w) {
		panel = draw.DebugPanel(w)
	}
	u := [16]float32{
		float32(view),
		scale,
		draw.LightMargin,
		float32(max(l.lastRows[win], 1)),
		rect.W,
		rect.H,
		fb.X,
		fb.Y,
		panel.X * scale,
		panel.Y * scale,
		(panel.X + panel.W) * scale,
		(panel.Y + panel.H) * scale,
		area.X * scale,
		area.Y * scale,
		(area.X + area.W) * scale,
		(area.Y + area.H) * scale,
	}
	r.dev.WriteBuffer(l.debugBuf, 0, unsafe.Pointer(&u[0]), 64)
	g := r.dev.CreateBindGroup(
		l.debugSet,
		wgpu.BindEntry{Binding: 0, Buffer: l.debugBuf, Size: 64},
		wgpu.BindEntry{Binding: 1, View: tex.View()},
	)
	pass.SetPipeline(pipe)
	pass.SetBindGroup(0, g)
	pass.SetScissor(0, 0, int(fb.X), int(fb.Y))
	pass.Draw(3, 0)
	l.debugGroups = append(l.debugGroups, g)
}

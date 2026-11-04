//go:build js

package wgpu

import (
	"fmt"
	"syscall/js"
)

type Stage uint64

const (
	StageVertex   Stage = 1
	StageFragment Stage = 2
)

type BindingKind int

const (
	BindUniform BindingKind = iota
	BindTexture
	BindSampler
	BindUnfilterable
	BindDepth
	BindDepthMultisampled
	BindSamplerNearest
	BindSamplerCompare
)

type LayoutEntry struct {
	Binding    int
	Visibility Stage
	Kind       BindingKind
	Dynamic    bool
}

type BindGroupLayout struct{ v js.Value }

func (d *Device) CreateBindGroupLayout(entries ...LayoutEntry) (*BindGroupLayout, error) {
	es := make([]any, len(entries))
	for i, e := range entries {
		o := object()
		o.Set("binding", e.Binding)
		o.Set("visibility", uint64(e.Visibility))
		switch e.Kind {
		case BindUniform:
			b := object()
			b.Set("type", "uniform")
			b.Set("hasDynamicOffset", e.Dynamic)
			o.Set("buffer", b)
		case BindTexture, BindUnfilterable, BindDepth, BindDepthMultisampled:
			t := object()
			t.Set("viewDimension", "2d")
			t.Set("sampleType", sampleTypeName(e.Kind))
			t.Set("multisampled", e.Kind == BindDepthMultisampled)
			o.Set("texture", t)
		case BindSampler, BindSamplerNearest, BindSamplerCompare:
			s := object()
			s.Set("type", samplerTypeName(e.Kind))
			o.Set("sampler", s)
		}
		es[i] = o
	}
	desc := object()
	desc.Set("entries", array(es...))
	var l js.Value
	if err := d.scoped(func() { l = d.v.Call("createBindGroupLayout", desc) }); err != nil {
		return nil, fmt.Errorf("wgpu: bind group layout: %w", err)
	}
	return &BindGroupLayout{v: l}, nil
}

func (l *BindGroupLayout) Release() {}

type BindEntry struct {
	Binding      int
	Buffer       *Buffer
	Offset, Size int
	View         TextureView
	Sampler      *Sampler
}

type BindGroup struct{ v js.Value }

func (d *Device) CreateBindGroup(layout *BindGroupLayout, entries ...BindEntry) *BindGroup {
	es := make([]any, len(entries))
	for i, e := range entries {
		o := object()
		o.Set("binding", e.Binding)
		switch {
		case e.Buffer != nil:
			b := object()
			b.Set("buffer", e.Buffer.v)
			b.Set("offset", e.Offset)
			b.Set("size", e.Size)
			o.Set("resource", b)
		case e.Sampler != nil:
			o.Set("resource", e.Sampler.v)
		default:
			o.Set("resource", e.View.js())
		}
		es[i] = o
	}
	desc := object()
	desc.Set("layout", layout.v)
	desc.Set("entries", array(es...))
	return &BindGroup{v: d.v.Call("createBindGroup", desc)}
}

func (g *BindGroup) Release() {}

type (
	BlendFactor uint32
	BlendOp     uint32
)

const (
	Zero BlendFactor = iota + 1
	One
	Src
	OneMinusSrc
	SrcAlpha
	OneMinusSrcAlpha
)

const (
	Add BlendOp = iota + 1
	Max
	ReverseSubtract
)

//nolint:gochecknoglobals // a lookup table
var factorNames = map[BlendFactor]string{
	Zero:             "zero",
	One:              "one",
	Src:              "src",
	OneMinusSrc:      "one-minus-src",
	SrcAlpha:         "src-alpha",
	OneMinusSrcAlpha: "one-minus-src-alpha",
}

//nolint:gochecknoglobals // a lookup table
var opNames = map[BlendOp]string{Add: "add", Max: "max", ReverseSubtract: "reverse-subtract"}

type BlendComponent struct {
	Op       BlendOp
	Src, Dst BlendFactor
}

func (c BlendComponent) js() js.Value {
	o := object()
	o.Set("operation", opNames[c.Op])
	o.Set("srcFactor", factorNames[c.Src])
	o.Set("dstFactor", factorNames[c.Dst])
	return o
}

type Blend struct{ Color, Alpha BlendComponent }

type VertexFormat uint32

const (
	Float32 VertexFormat = iota + 1
	Float32x2
	Float32x3
	Float32x4
)

//nolint:gochecknoglobals // a lookup table
var vertexFormatNames = map[VertexFormat]string{
	Float32:   "float32",
	Float32x2: "float32x2",
	Float32x3: "float32x3",
	Float32x4: "float32x4",
}

type Topology int

const (
	Triangles Topology = iota
	Lines
)

type Cull int

const (
	CullNone Cull = iota
	CullBack
)

type DepthState struct {
	Format    TextureFormat
	Write     bool
	LessEqual bool
	Always    bool
	Bias      int
	SlopeBias float32
}

type VertexBuffer struct {
	Stride     uintptr
	Instance   bool
	Attributes []Attribute
}

type Attribute struct {
	Location int
	Format   VertexFormat
	Offset   uintptr
}

type PipelineSpec struct {
	Label      string
	Layouts    []*BindGroupLayout
	Shader     *Shader
	Vertex     string
	Fragment   string
	Stride     uintptr
	Attributes []Attribute
	Buffers    []VertexBuffer
	Format     TextureFormat
	Blend      *Blend
	Topology   Topology
	Cull       Cull
	Depth      *DepthState
	Samples    int
}

type RenderPipeline struct{ v js.Value }

func (d *Device) CreateRenderPipeline(s PipelineSpec) (*RenderPipeline, error) {
	layouts := make([]any, len(s.Layouts))
	for i, l := range s.Layouts {
		layouts[i] = l.v
	}
	pl := object()
	pl.Set("bindGroupLayouts", array(layouts...))
	desc := object()
	desc.Set("label", s.Label)
	desc.Set("layout", d.v.Call("createPipelineLayout", pl))

	desc.Set("vertex", vertexState(s))

	prim := object()
	prim.Set("topology", map[Topology]string{Triangles: "triangle-list", Lines: "line-list"}[s.Topology])
	prim.Set("cullMode", map[Cull]string{CullNone: "none", CullBack: "back"}[s.Cull])
	prim.Set("frontFace", "ccw")
	desc.Set("primitive", prim)
	ms := object()
	ms.Set("count", max(s.Samples, 1))
	desc.Set("multisample", ms)

	if s.Depth != nil {
		desc.Set("depthStencil", depthStencilState(s.Depth))
	}
	if s.Fragment != "" {
		desc.Set("fragment", fragmentState(s))
	}
	var p js.Value
	if err := d.scoped(func() { p = d.v.Call("createRenderPipeline", desc) }); err != nil {
		return nil, fmt.Errorf("wgpu: pipeline %s: %w", s.Label, err)
	}
	return &RenderPipeline{v: p}, nil
}

// vertexState is s's vertex stage: its shader entry and buffers.
func vertexState(s PipelineSpec) js.Value {
	vertex := object()
	vertex.Set("module", s.Shader.v)
	vertex.Set("entryPoint", s.Vertex)
	buffers := s.Buffers
	if len(buffers) == 0 && len(s.Attributes) > 0 {
		buffers = []VertexBuffer{{Stride: s.Stride, Attributes: s.Attributes}}
	}
	vbs := make([]any, len(buffers))
	for i, b := range buffers {
		attrs := make([]any, len(b.Attributes))
		for j, at := range b.Attributes {
			a := object()
			a.Set("format", vertexFormatNames[at.Format])
			a.Set("offset", int(at.Offset))
			a.Set("shaderLocation", at.Location)
			attrs[j] = a
		}
		vb := object()
		vb.Set("arrayStride", int(b.Stride))
		vb.Set("stepMode", map[bool]string{false: "vertex", true: "instance"}[b.Instance])
		vb.Set("attributes", array(attrs...))
		vbs[i] = vb
	}
	vertex.Set("buffers", array(vbs...))
	return vertex
}

// depthStencilState is d's depth test.
func depthStencilState(d *DepthState) js.Value {
	ds := object()
	ds.Set("format", d.Format.String())
	ds.Set("depthWriteEnabled", d.Write)
	compare := "less"
	switch {
	case d.Always:
		compare = "always"
	case d.LessEqual:
		compare = "less-equal"
	}
	ds.Set("depthCompare", compare)
	ds.Set("depthBias", d.Bias)
	ds.Set("depthBiasSlopeScale", d.SlopeBias)
	return ds
}

// fragmentState is s's fragment stage: its shader entry and, when s has a
// format, its one target with its blend.
func fragmentState(s PipelineSpec) js.Value {
	frag := object()
	frag.Set("module", s.Shader.v)
	frag.Set("entryPoint", s.Fragment)
	var targets []any
	if s.Format != 0 {
		t := object()
		t.Set("format", s.Format.String())
		if s.Blend != nil {
			b := object()
			b.Set("color", s.Blend.Color.js())
			b.Set("alpha", s.Blend.Alpha.js())
			t.Set("blend", b)
		}
		targets = append(targets, t)
	}
	frag.Set("targets", array(targets...))
	return frag
}

// sampleTypeName is WebGPU's name for what a texture binding of kind k
// samples; empty for a binding that is no texture.
func sampleTypeName(k BindingKind) string {
	switch k {
	case BindTexture:
		return "float"
	case BindUnfilterable:
		return "unfilterable-float"
	case BindDepth, BindDepthMultisampled:
		return "depth"
	case BindUniform, BindSampler, BindSamplerNearest, BindSamplerCompare:
	}
	return ""
}

// samplerTypeName is WebGPU's name for a sampler binding of kind k; empty
// for a binding that is no sampler.
func samplerTypeName(k BindingKind) string {
	switch k {
	case BindSampler:
		return "filtering"
	case BindSamplerNearest:
		return "non-filtering"
	case BindSamplerCompare:
		return "comparison"
	case BindUniform, BindTexture, BindUnfilterable, BindDepth, BindDepthMultisampled:
	}
	return ""
}

func (p *RenderPipeline) Release() {}

type Encoder struct {
	v js.Value
	d *Device
}

func (d *Device) CreateEncoder() *Encoder {
	return &Encoder{v: d.v.Call("createCommandEncoder"), d: d}
}

type Color struct{ R, G, B, A float64 }

type Pass struct{ v js.Value }

func (e *Encoder) BeginPass(view TextureView, clearTo *Color, _ ...Span) *Pass {
	return e.BeginPassSpec(PassSpec{Color: view, Clear: clearTo})
}

type PassSpec struct {
	Color, Resolve TextureView
	Clear          *Color
	Depth          TextureView
	DepthClear     float32
	KeepDepth      bool
	Span           Span
}

func (e *Encoder) BeginPassSpec(s PassSpec) *Pass {
	desc := object()
	var colors []any
	if s.Color.p != nil {
		att := object()
		att.Set("view", s.Color.js())
		if s.Resolve.p != nil {
			att.Set("resolveTarget", s.Resolve.js())
		}
		att.Set("storeOp", "store")
		att.Set("loadOp", "load")
		if s.Clear != nil {
			att.Set("loadOp", "clear")
			c := object()
			c.Set("r", s.Clear.R)
			c.Set("g", s.Clear.G)
			c.Set("b", s.Clear.B)
			c.Set("a", s.Clear.A)
			att.Set("clearValue", c)
		}
		colors = append(colors, att)
	}
	desc.Set("colorAttachments", array(colors...))
	if s.Depth.p != nil {
		ds := object()
		ds.Set("view", s.Depth.js())
		ds.Set("depthStoreOp", "store")
		ds.Set("depthLoadOp", "clear")
		ds.Set("depthClearValue", s.DepthClear)
		if s.KeepDepth {
			ds.Set("depthLoadOp", "load")
		}
		desc.Set("depthStencilAttachment", ds)
	}
	return &Pass{v: e.v.Call("beginRenderPass", desc)}
}

func (p *Pass) SetPipeline(pl *RenderPipeline) { p.v.Call("setPipeline", pl.v) }

func (p *Pass) SetBindGroup(i int, g *BindGroup, offsets ...uint32) {
	if len(offsets) == 0 {
		p.v.Call("setBindGroup", i, g.v)
		return
	}
	os := make([]any, len(offsets))
	for k, o := range offsets {
		os[k] = o
	}
	p.v.Call("setBindGroup", i, g.v, array(os...))
}

func (p *Pass) SetVertexBuffer(b *Buffer, offset, size int) {
	p.SetVertexBufferSlot(0, b, offset, size)
}

func (p *Pass) SetVertexBufferSlot(slot int, b *Buffer, offset, size int) {
	p.v.Call("setVertexBuffer", slot, b.v, offset, size)
}

func (p *Pass) SetIndexBuffer(b *Buffer, offset, size int) {
	p.v.Call("setIndexBuffer", b.v, "uint32", offset, size)
}

func (p *Pass) SetScissor(x, y, w, h int) { p.v.Call("setScissorRect", x, y, w, h) }

func (p *Pass) SetViewport(x, y, w, h float32) { p.v.Call("setViewport", x, y, w, h, 0, 1) }

func (p *Pass) Draw(vertices, first int) { p.v.Call("draw", vertices, 1, first, 0) }

func (p *Pass) DrawIndexed(count, first int) { p.DrawIndexedInstanced(count, first, 1) }

func (p *Pass) DrawInstanced(vertices, instances int) { p.v.Call("draw", vertices, instances, 0, 0) }

func (p *Pass) DrawIndexedInstanced(count, first, instances int) {
	p.v.Call("drawIndexed", count, instances, first, 0, 0)
}

func (p *Pass) End() { p.v.Call("end") }

func (e *Encoder) Submit() { e.d.q.Call("submit", array(e.v.Call("finish"))) }

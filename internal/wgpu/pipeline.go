package wgpu

/*
#include <stdlib.h>
#include "procs.h"
*/
import "C"

import "fmt"

// Stage says which shader stages see a binding.
type Stage uint64

// BindingKind is what a bind group layout's entry holds.
type BindingKind int

const (
	BindUniform BindingKind = iota
	BindTexture
	BindSampler
	// BindUnfilterable is a float texture read without filtering (R32Float);
	// BindDepth a depth texture, read with a comparison sampler or not;
	// BindDepthMultisampled a multisampled depth texture, read by sample.
	BindUnfilterable
	BindDepth
	BindDepthMultisampled
	// BindSamplerNearest reads without filtering; BindSamplerCompare
	// compares depth.
	BindSamplerNearest
	BindSamplerCompare
)

// LayoutEntry is one binding of a bind group layout. A uniform with
// Dynamic takes its offset when the group is set.
type LayoutEntry struct {
	Binding    int
	Visibility Stage
	Kind       BindingKind
	Dynamic    bool
}

// BindGroupLayout is the shape of a bind group.
type BindGroupLayout struct{ h C.WGPUBindGroupLayout }

func (d *Device) CreateBindGroupLayout(entries ...LayoutEntry) (*BindGroupLayout, error) {
	var a arena
	defer a.free()
	es := allocN[C.WGPUBindGroupLayoutEntry](&a, len(entries))
	for i, e := range entries {
		es[i].binding = C.uint32_t(e.Binding)
		es[i].visibility = C.WGPUShaderStage(e.Visibility)
		switch e.Kind {
		case BindUniform:
			es[i].buffer._type = C.WGPUBufferBindingType_Uniform
			if e.Dynamic {
				es[i].buffer.hasDynamicOffset = 1
			}
		case BindTexture, BindUnfilterable, BindDepth, BindDepthMultisampled:
			es[i].texture.viewDimension = C.WGPUTextureViewDimension_2D
			es[i].texture.sampleType = sampleType(e.Kind)
			if e.Kind == BindDepthMultisampled {
				es[i].texture.multisampled = 1
			}
		case BindSampler:
			es[i].sampler._type = C.WGPUSamplerBindingType_Filtering
		case BindSamplerNearest:
			es[i].sampler._type = C.WGPUSamplerBindingType_NonFiltering
		case BindSamplerCompare:
			es[i].sampler._type = C.WGPUSamplerBindingType_Comparison
		}
	}
	desc := alloc[C.WGPUBindGroupLayoutDescriptor](&a)
	desc.entryCount = C.size_t(len(entries))
	if len(entries) > 0 {
		desc.entries = &es[0]
	}
	var h C.WGPUBindGroupLayout
	if err := d.scoped(func() { h = C.e_wgpuDeviceCreateBindGroupLayout(d.h, desc) }); err != nil {
		return nil, fmt.Errorf("wgpu: bind group layout: %w", err)
	}
	return &BindGroupLayout{h: h}, nil
}

func (l *BindGroupLayout) Release() { C.e_wgpuBindGroupLayoutRelease(l.h) }

// BindEntry is what one binding of a bind group holds: a buffer's range,
// a texture view or a sampler.
type BindEntry struct {
	Binding      int
	Buffer       *Buffer
	Offset, Size int
	View         TextureView
	Sampler      *Sampler
}

// BindGroup is the resources a draw reads, set together.
type BindGroup struct{ h C.WGPUBindGroup }

func (d *Device) CreateBindGroup(layout *BindGroupLayout, entries ...BindEntry) *BindGroup {
	var a arena
	defer a.free()
	es := allocN[C.WGPUBindGroupEntry](&a, len(entries))
	for i, e := range entries {
		es[i].binding = C.uint32_t(e.Binding)
		switch {
		case e.Buffer != nil:
			es[i].buffer = e.Buffer.h
			es[i].offset, es[i].size = C.uint64_t(e.Offset), C.uint64_t(e.Size)
		case e.Sampler != nil:
			es[i].sampler = e.Sampler.h
		default:
			es[i].textureView = e.View.h
		}
	}
	desc := alloc[C.WGPUBindGroupDescriptor](&a)
	desc.layout = layout.h
	desc.entryCount = C.size_t(len(entries))
	if len(entries) > 0 {
		desc.entries = &es[0]
	}
	return &BindGroup{h: C.e_wgpuDeviceCreateBindGroup(d.h, desc)}
}

func (g *BindGroup) Release() { C.e_wgpuBindGroupRelease(g.h) }

// BlendFactor and BlendOp say how a draw's colour combines with the
// target's.
type (
	BlendFactor uint32
	BlendOp     uint32
)

const (
	Zero             BlendFactor = C.WGPUBlendFactor_Zero
	One              BlendFactor = C.WGPUBlendFactor_One
	Src              BlendFactor = C.WGPUBlendFactor_Src
	OneMinusSrc      BlendFactor = C.WGPUBlendFactor_OneMinusSrc
	SrcAlpha         BlendFactor = C.WGPUBlendFactor_SrcAlpha
	OneMinusSrcAlpha BlendFactor = C.WGPUBlendFactor_OneMinusSrcAlpha

	Add             BlendOp = C.WGPUBlendOperation_Add
	Max             BlendOp = C.WGPUBlendOperation_Max
	ReverseSubtract BlendOp = C.WGPUBlendOperation_ReverseSubtract
)

// BlendComponent is one channel group's blend: Src×Op×Dst.
type BlendComponent struct {
	Op       BlendOp
	Src, Dst BlendFactor
}

// Blend is a pipeline's blend state; nil draws replace the target.
type Blend struct{ Color, Alpha BlendComponent }

// VertexFormat is the type of one vertex attribute.
type VertexFormat uint32

const (
	Float32   VertexFormat = C.WGPUVertexFormat_Float32
	Float32x2 VertexFormat = C.WGPUVertexFormat_Float32x2
	Float32x3 VertexFormat = C.WGPUVertexFormat_Float32x3
	Float32x4 VertexFormat = C.WGPUVertexFormat_Float32x4
)

// Topology is how a pipeline's vertices make primitives.
type Topology int

const (
	Triangles Topology = iota
	Lines
)

// Cull is which faces a pipeline leaves out.
type Cull int

const (
	CullNone Cull = iota
	CullBack
)

// DepthState is how a pipeline tests and writes depth, and its bias.
type DepthState struct {
	Format    TextureFormat
	Write     bool
	LessEqual bool
	// Always passes every fragment, for what draws over the depth.
	Always    bool
	Bias      int
	SlopeBias float32
}

// VertexBuffer is one vertex buffer's layout: per vertex, or per instance.
type VertexBuffer struct {
	Stride     uintptr
	Instance   bool
	Attributes []Attribute
}

// Attribute is one vertex attribute: its location, format and offset.
type Attribute struct {
	Location int
	Format   VertexFormat
	Offset   uintptr
}

// PipelineSpec is everything a render pipeline is made from: its bind
// group layouts, its shader and entry points, its vertex layout, the
// format it draws into and its blend.
type PipelineSpec struct {
	Label      string
	Layouts    []*BindGroupLayout
	Shader     *Shader
	Vertex     string
	Fragment   string
	Stride     uintptr
	Attributes []Attribute
	// Buffers, when set, are the vertex buffers instead of Stride and
	// Attributes, one per slot.
	Buffers []VertexBuffer
	// Format is what it draws into, none for a depth-only pipeline.
	Format   TextureFormat
	Blend    *Blend
	Topology Topology
	Cull     Cull
	Depth    *DepthState
	Samples  int
}

// RenderPipeline is a compiled draw state.
type RenderPipeline struct {
	h      C.WGPURenderPipeline
	layout C.WGPUPipelineLayout
}

func (d *Device) CreateRenderPipeline(s PipelineSpec) (*RenderPipeline, error) {
	var a arena
	defer a.free()
	layouts := allocN[C.WGPUBindGroupLayout](&a, len(s.Layouts))
	for i, l := range s.Layouts {
		layouts[i] = l.h
	}
	pl := alloc[C.WGPUPipelineLayoutDescriptor](&a)
	pl.bindGroupLayoutCount = C.size_t(len(layouts))
	if len(layouts) > 0 {
		pl.bindGroupLayouts = &layouts[0]
	}
	layout := C.e_wgpuDeviceCreatePipelineLayout(d.h, pl)

	desc := alloc[C.WGPURenderPipelineDescriptor](&a)
	desc.label = a.str(s.Label)
	desc.layout = layout
	desc.vertex.module = s.Shader.h
	desc.vertex.entryPoint = a.str(s.Vertex)
	buffers := s.Buffers
	if len(buffers) == 0 && len(s.Attributes) > 0 {
		buffers = []VertexBuffer{{Stride: s.Stride, Attributes: s.Attributes}}
	}
	desc.vertex.buffers, desc.vertex.bufferCount = vertexBuffers(&a, buffers)
	desc.primitive.topology = C.WGPUPrimitiveTopology_TriangleList
	if s.Topology == Lines {
		desc.primitive.topology = C.WGPUPrimitiveTopology_LineList
	}
	desc.primitive.cullMode = C.WGPUCullMode_None
	if s.Cull == CullBack {
		desc.primitive.cullMode = C.WGPUCullMode_Back
	}
	desc.primitive.frontFace = C.WGPUFrontFace_CCW
	desc.multisample.count = C.uint32_t(max(s.Samples, 1))
	desc.multisample.mask = 0xFFFFFFFF
	if s.Depth != nil {
		desc.depthStencil = depthStencil(&a, s.Depth)
	}
	target := alloc[C.WGPUColorTargetState](&a)
	target.format = C.WGPUTextureFormat(s.Format)
	target.writeMask = colorWriteAll
	if s.Blend != nil {
		target.blend = blendState(&a, s.Blend)
	}
	if s.Fragment != "" {
		frag := alloc[C.WGPUFragmentState](&a)
		frag.module = s.Shader.h
		frag.entryPoint = a.str(s.Fragment)
		if s.Format != 0 {
			frag.targetCount = 1
			frag.targets = target
		}
		desc.fragment = frag
	}
	var h C.WGPURenderPipeline
	if err := d.scoped(func() { h = C.e_wgpuDeviceCreateRenderPipeline(d.h, desc) }); err != nil {
		C.e_wgpuPipelineLayoutRelease(layout)
		return nil, fmt.Errorf("wgpu: pipeline %s: %w", s.Label, err)
	}
	return &RenderPipeline{h: h, layout: layout}, nil
}

// sampleType is what a texture binding of kind k samples; nothing for a
// binding that is no texture.
func sampleType(k BindingKind) C.WGPUTextureSampleType {
	switch k {
	case BindTexture:
		return C.WGPUTextureSampleType_Float
	case BindUnfilterable:
		return C.WGPUTextureSampleType_UnfilterableFloat
	case BindDepth, BindDepthMultisampled:
		return C.WGPUTextureSampleType_Depth
	case BindUniform, BindSampler, BindSamplerNearest, BindSamplerCompare:
	}
	return 0
}

// vertexBuffers lays out buffers for a pipeline, in a.
func vertexBuffers(a *arena, buffers []VertexBuffer) (*C.WGPUVertexBufferLayout, C.size_t) {
	if len(buffers) == 0 {
		return nil, 0
	}
	vbs := allocN[C.WGPUVertexBufferLayout](a, len(buffers))
	for i, b := range buffers {
		attrs := allocN[C.WGPUVertexAttribute](a, len(b.Attributes))
		for j, at := range b.Attributes {
			attrs[j].format = C.WGPUVertexFormat(at.Format)
			attrs[j].offset = C.uint64_t(at.Offset)
			attrs[j].shaderLocation = C.uint32_t(at.Location)
		}
		vbs[i].stepMode = C.WGPUVertexStepMode_Vertex
		if b.Instance {
			vbs[i].stepMode = C.WGPUVertexStepMode_Instance
		}
		vbs[i].arrayStride = C.uint64_t(b.Stride)
		vbs[i].attributeCount = C.size_t(len(attrs))
		if len(attrs) > 0 {
			vbs[i].attributes = &attrs[0]
		}
	}
	return &vbs[0], C.size_t(len(vbs))
}

// depthStencil is d's depth test, in a, with the stencil always passing.
func depthStencil(a *arena, d *DepthState) *C.WGPUDepthStencilState {
	ds := alloc[C.WGPUDepthStencilState](a)
	ds.format = C.WGPUTextureFormat(d.Format)
	ds.depthWriteEnabled = C.WGPUOptionalBool_False
	if d.Write {
		ds.depthWriteEnabled = C.WGPUOptionalBool_True
	}
	ds.depthCompare = C.WGPUCompareFunction_Less
	switch {
	case d.Always:
		ds.depthCompare = C.WGPUCompareFunction_Always
	case d.LessEqual:
		ds.depthCompare = C.WGPUCompareFunction_LessEqual
	}
	for _, f := range []*C.WGPUStencilFaceState{&ds.stencilFront, &ds.stencilBack} {
		f.compare = C.WGPUCompareFunction_Always
		keep := C.WGPUStencilOperation(C.WGPUStencilOperation_Keep)
		f.failOp, f.depthFailOp, f.passOp = keep, keep, keep
	}
	ds.depthBias = C.int32_t(d.Bias)
	ds.depthBiasSlopeScale = C.float(d.SlopeBias)
	return ds
}

// blendState is bl, in a.
func blendState(a *arena, bl *Blend) *C.WGPUBlendState {
	b := alloc[C.WGPUBlendState](a)
	b.color = C.WGPUBlendComponent{
		operation: C.WGPUBlendOperation(bl.Color.Op),
		srcFactor: C.WGPUBlendFactor(bl.Color.Src),
		dstFactor: C.WGPUBlendFactor(bl.Color.Dst),
	}
	b.alpha = C.WGPUBlendComponent{
		operation: C.WGPUBlendOperation(bl.Alpha.Op),
		srcFactor: C.WGPUBlendFactor(bl.Alpha.Src),
		dstFactor: C.WGPUBlendFactor(bl.Alpha.Dst),
	}
	return b
}

func (p *RenderPipeline) Release() {
	C.e_wgpuRenderPipelineRelease(p.h)
	C.e_wgpuPipelineLayoutRelease(p.layout)
}

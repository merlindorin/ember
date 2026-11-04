package wgpu

/*
#include <stdlib.h>
#include <string.h>
#include "procs.h"
#include "callbacks.h"
*/
import "C"

import (
	"fmt"
	"image"
	"unsafe"
)

// Encoder records one batch of GPU commands.
type Encoder struct {
	h C.WGPUCommandEncoder
	d *Device
}

func (d *Device) CreateEncoder() *Encoder {
	return &Encoder{h: C.e_wgpuDeviceCreateCommandEncoder(d.h, nil), d: d}
}

// Color is a clear colour.
type Color struct{ R, G, B, A float64 }

// Pass is a render pass, drawing into one texture.
type Pass struct{ h C.WGPURenderPassEncoder }

// BeginPass begins a pass into view: cleared to clearTo, or, when clearTo is
// nil, keeping what the view holds; timed by span, when there is one.
func (e *Encoder) BeginPass(view TextureView, clearTo *Color, span ...Span) *Pass {
	s := PassSpec{Color: view, Clear: clearTo}
	if len(span) > 0 {
		s.Span = span[0]
	}
	return e.BeginPassSpec(s)
}

// PassSpec is what a pass draws into: a colour view, none for a
// depth-only pass, resolved into Resolve when it is multisampled; a depth
// view; how each starts (cleared, or as it is when Clear is nil or
// KeepDepth); and its timestamps.
type PassSpec struct {
	Color, Resolve TextureView
	Clear          *Color
	Depth          TextureView
	DepthClear     float32
	KeepDepth      bool
	Span           Span
}

// BeginPassSpec begins the pass s describes.
func (e *Encoder) BeginPassSpec(s PassSpec) *Pass {
	var a arena
	defer a.free()
	desc := alloc[C.WGPURenderPassDescriptor](&a)
	if s.Color.h != nil {
		att := alloc[C.WGPURenderPassColorAttachment](&a)
		att.view = s.Color.h
		att.resolveTarget = s.Resolve.h
		att.depthSlice = C.WGPU_DEPTH_SLICE_UNDEFINED
		att.storeOp = C.WGPUStoreOp_Store
		att.loadOp = C.WGPULoadOp_Load
		if s.Clear != nil {
			att.loadOp = C.WGPULoadOp_Clear
			att.clearValue = C.WGPUColor{
				r: C.double(s.Clear.R),
				g: C.double(s.Clear.G),
				b: C.double(s.Clear.B),
				a: C.double(s.Clear.A),
			}
		}
		desc.colorAttachmentCount = 1
		desc.colorAttachments = att
	}
	if s.Depth.h != nil {
		d := alloc[C.WGPURenderPassDepthStencilAttachment](&a)
		d.view = s.Depth.h
		d.depthStoreOp = C.WGPUStoreOp_Store
		d.depthLoadOp = C.WGPULoadOp_Clear
		d.depthClearValue = C.float(s.DepthClear)
		if s.KeepDepth {
			d.depthLoadOp = C.WGPULoadOp_Load
		}
		desc.depthStencilAttachment = d
	}
	if s.Span.set != nil {
		tw := alloc[C.WGPUPassTimestampWrites](&a)
		tw.querySet = s.Span.set
		tw.beginningOfPassWriteIndex = C.uint32_t(s.Span.begin)
		tw.endOfPassWriteIndex = C.uint32_t(s.Span.begin + 1)
		desc.timestampWrites = tw
	}
	return &Pass{h: C.e_wgpuCommandEncoderBeginRenderPass(e.h, desc)}
}

func (p *Pass) SetPipeline(pl *RenderPipeline) { C.e_wgpuRenderPassEncoderSetPipeline(p.h, pl.h) }

// SetBindGroup sets group i, with the offsets of its dynamic uniforms.
func (p *Pass) SetBindGroup(i int, g *BindGroup, offsets ...uint32) {
	var ptr *C.uint32_t
	if len(offsets) > 0 {
		ptr = (*C.uint32_t)(unsafe.Pointer(&offsets[0]))
	}
	C.e_wgpuRenderPassEncoderSetBindGroup(p.h, C.uint32_t(i), g.h, C.size_t(len(offsets)), ptr)
}

func (p *Pass) SetVertexBuffer(b *Buffer, offset, size int) {
	p.SetVertexBufferSlot(0, b, offset, size)
}

// SetVertexBufferSlot sets the vertex buffer of slot.
func (p *Pass) SetVertexBufferSlot(slot int, b *Buffer, offset, size int) {
	C.e_wgpuRenderPassEncoderSetVertexBuffer(p.h, C.uint32_t(slot), b.h, C.uint64_t(offset), C.uint64_t(size))
}

func (p *Pass) SetIndexBuffer(b *Buffer, offset, size int) {
	C.e_wgpuRenderPassEncoderSetIndexBuffer(p.h, b.h, C.WGPUIndexFormat_Uint32, C.uint64_t(offset), C.uint64_t(size))
}

func (p *Pass) SetScissor(x, y, w, h int) {
	C.e_wgpuRenderPassEncoderSetScissorRect(p.h, C.uint32_t(x), C.uint32_t(y), C.uint32_t(w), C.uint32_t(h))
}

func (p *Pass) SetViewport(x, y, w, h float32) {
	C.e_wgpuRenderPassEncoderSetViewport(p.h, C.float(x), C.float(y), C.float(w), C.float(h), 0, 1)
}

func (p *Pass) Draw(vertices, first int) {
	C.e_wgpuRenderPassEncoderDraw(p.h, C.uint32_t(vertices), 1, C.uint32_t(first), 0)
}

func (p *Pass) DrawIndexed(count, first int) { p.DrawIndexedInstanced(count, first, 1) }

// DrawInstanced draws vertices vertices, instances times.
func (p *Pass) DrawInstanced(vertices, instances int) {
	C.e_wgpuRenderPassEncoderDraw(p.h, C.uint32_t(vertices), C.uint32_t(instances), 0, 0)
}

// DrawIndexedInstanced draws count indices from first, instances times.
func (p *Pass) DrawIndexedInstanced(count, first, instances int) {
	C.e_wgpuRenderPassEncoderDrawIndexed(p.h, C.uint32_t(count), C.uint32_t(instances), C.uint32_t(first), 0, 0)
}

// End ends the pass and lets it go.
func (p *Pass) End() {
	C.e_wgpuRenderPassEncoderEnd(p.h)
	C.e_wgpuRenderPassEncoderRelease(p.h)
}

// Submit finishes e's commands and runs them.
func (e *Encoder) Submit() {
	cmd := C.e_wgpuCommandEncoderFinish(e.h, nil)
	C.e_wgpuQueueSubmit(e.d.q, 1, &cmd) //nolint:gocritic // cgo's own pointer check on &cmd, not this line
	C.e_wgpuCommandBufferRelease(cmd)
	C.e_wgpuCommandEncoderRelease(e.h)
}

// Read copies t back from the GPU, waiting for it, as an opaque RGBA
// image, top row first: a capture of what was drawn.
func (d *Device) Read(t *Texture) (*image.RGBA, error) {
	row := (t.Width*4 + 255) &^ 255
	size := row * t.Height
	buf := d.CreateBuffer(size, BufferCopyDst|BufferMapRead)
	defer buf.Release()
	var a arena
	defer a.free()
	src := alloc[C.WGPUTexelCopyTextureInfo](&a)
	src.texture = t.h
	src.aspect = C.WGPUTextureAspect_All
	dst := alloc[C.WGPUTexelCopyBufferInfo](&a)
	dst.buffer = buf.h
	dst.layout.bytesPerRow = C.uint32_t(row)
	dst.layout.rowsPerImage = C.uint32_t(t.Height)
	extent := alloc[C.WGPUExtent3D](&a)
	extent.width, extent.height, extent.depthOrArrayLayers = C.uint32_t(t.Width), C.uint32_t(t.Height), 1
	e := d.CreateEncoder()
	C.e_wgpuCommandEncoderCopyTextureToBuffer(e.h, src, dst, extent)
	e.Submit()
	w := newWait()
	defer freeWait(w)
	C.ember_map_read(buf.h, C.size_t(size), w)
	for w.done == 0 {
		C.e_wgpuDevicePoll(d.h, 1, nil)
		C.e_wgpuInstanceProcessEvents(d.inst.h)
	}
	if w.status != C.WGPUMapAsyncStatus_Success {
		return nil, fmt.Errorf("wgpu: read back: %s", C.GoString(&w.message[0]))
	}
	defer C.e_wgpuBufferUnmap(buf.h)
	mapped := unsafe.Slice((*byte)(C.e_wgpuBufferGetConstMappedRange(buf.h, 0, C.size_t(size))), size)
	img := image.NewRGBA(image.Rect(0, 0, t.Width, t.Height))
	bgra := t.Format == BGRA8Unorm
	for y := range t.Height {
		in, out := mapped[y*row:y*row+t.Width*4], img.Pix[y*img.Stride:y*img.Stride+t.Width*4]
		copy(out, in)
		for x := 0; x < len(out); x += 4 {
			if bgra {
				out[x], out[x+2] = out[x+2], out[x]
			}
			out[x+3] = 255
		}
	}
	return img, nil
}

// Poll waits for the GPU to finish what was submitted, and runs the
// callbacks that are due.
func (d *Device) Poll() {
	C.e_wgpuDevicePoll(d.h, 1, nil)
	C.e_wgpuInstanceProcessEvents(d.inst.h)
}

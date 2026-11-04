package wgpu

/*
#include <stdlib.h>
#include "procs.h"
#include "callbacks.h"
*/
import "C"

import "unsafe"

// readbackSlots is how many copies a Readback has in flight at once.
const readbackSlots = 4

// Readback copies a small texture back from the GPU without waiting: a
// frame's copy is read once the GPU has done with it, a few frames later.
// Each copy carries a tag, given back with its bytes.
type Readback struct {
	d     *Device
	slots [readbackSlots]readbackSlot
	next  int
}

type readbackSlot struct {
	buf     *Buffer
	wait    *C.ember_wait
	row     int
	height  int
	pending bool
	mapped  bool
	tag     any
}

// NewReadback is a readback on d.
func (d *Device) NewReadback() *Readback { return &Readback{d: d} }

// Copy has e copy t into a free slot, tagged tag, and reports whether one
// was free; Submitted then starts reading it.
func (r *Readback) Copy(e *Encoder, t *Texture, bytesPerTexel int, tag any) bool {
	s := &r.slots[r.next]
	if s.pending {
		return false
	}
	row := (t.Width*bytesPerTexel + 255) &^ 255
	size := row * t.Height
	if s.buf == nil || s.buf.Size < size {
		if s.buf != nil {
			s.buf.Release()
		}
		s.buf = r.d.CreateBuffer(size, BufferCopyDst|BufferMapRead)
	}
	if s.wait == nil {
		s.wait = newWait()
	}
	var a arena
	defer a.free()
	src := alloc[C.WGPUTexelCopyTextureInfo](&a)
	src.texture = t.h
	src.aspect = C.WGPUTextureAspect_All
	dst := alloc[C.WGPUTexelCopyBufferInfo](&a)
	dst.buffer = s.buf.h
	dst.layout.bytesPerRow = C.uint32_t(row)
	dst.layout.rowsPerImage = C.uint32_t(t.Height)
	extent := alloc[C.WGPUExtent3D](&a)
	extent.width, extent.height, extent.depthOrArrayLayers = C.uint32_t(t.Width), C.uint32_t(t.Height), 1
	C.e_wgpuCommandEncoderCopyTextureToBuffer(e.h, src, dst, extent)
	s.row, s.height, s.tag, s.mapped = row, t.Height, tag, true
	return true
}

// Submitted starts reading the copy Copy recorded, once its commands are
// submitted.
func (r *Readback) Submitted() {
	s := &r.slots[r.next]
	if !s.mapped || s.pending {
		return
	}
	*s.wait = C.ember_wait{}
	C.ember_map_read(s.buf.h, C.size_t(s.row*s.height), s.wait)
	s.pending, s.mapped = true, false
	r.next = (r.next + 1) % readbackSlots
}

// Done hands each copy that is back to read, oldest first: its rows of
// row bytes and its tag. The bytes are only valid during read.
func (r *Readback) Done(read func(rows []byte, row int, tag any)) {
	C.e_wgpuInstanceProcessEvents(r.d.inst.h)
	for k := range readbackSlots {
		s := &r.slots[(r.next+k)%readbackSlots]
		if !s.pending || s.wait.done == 0 {
			continue
		}
		s.pending = false
		if s.wait.status != C.WGPUMapAsyncStatus_Success {
			continue
		}
		size := s.row * s.height
		read(unsafe.Slice((*byte)(C.e_wgpuBufferGetConstMappedRange(s.buf.h, 0, C.size_t(size))), size), s.row, s.tag)
		C.e_wgpuBufferUnmap(s.buf.h)
	}
}

package wgpu

/*
#include <stdlib.h>
#include "procs.h"
#include "callbacks.h"
*/
import "C"

import (
	"encoding/binary"
	"math"
	"time"
	"unsafe"
)

// timerFrames is how many frames a timer's stamps wait before they are
// read, so reading them never stalls the GPU.
const timerFrames = 4

// FrameSpan is the name Frame reports the frame's whole GPU time under,
// from the first pass's beginning to the last one's end.
const FrameSpan = "frame"

// timerSpans is the most passes a frame times.
const timerSpans = 64

// Timers measure how long the GPU takes over passes, by the timestamps it
// writes as each begins and ends, read back timerFrames frames later. A
// device without timestamp queries times nothing. Passes are only timed
// while On.
type Timers struct {
	On     bool
	d      *Device
	slots  [timerFrames]timerSlot
	slot   int
	period float64
}

type timerSlot struct {
	set           C.WGPUQuerySet
	resolve, read *Buffer
	names         []string
	wait          *C.ember_wait
	pending       bool
}

// Span is where a pass writes its timestamps; the zero Span times nothing.
type Span struct {
	set   C.WGPUQuerySet
	begin int
}

// NewTimers is d's timers, which time nothing when d has no timestamps.
func (d *Device) NewTimers() *Timers {
	return &Timers{d: d, period: float64(C.e_wgpuQueueGetTimestampPeriod(d.q))}
}

// Frame reports through done the spans of the frame timerFrames ago whose
// stamps are back, and makes room for this frame's. A frame whose slot is
// still waiting for its last stamps is not timed.
func (t *Timers) Frame(done func(name string, d time.Duration)) {
	t.slot = (t.slot + 1) % timerFrames
	s := &t.slots[t.slot]
	if s.pending {
		C.e_wgpuInstanceProcessEvents(t.d.inst.h)
		if s.wait.done == 0 {
			return
		}
		s.pending = false
		if s.wait.status == C.WGPUMapAsyncStatus_Success {
			t.report(s, done)
		}
	}
	s.names = s.names[:0]
}

// report reads slot s's mapped stamps back through done, then unmaps them.
func (t *Timers) report(s *timerSlot, done func(name string, d time.Duration)) {
	size := len(s.names) * 16
	raw := unsafe.Slice((*byte)(C.e_wgpuBufferGetConstMappedRange(s.read.h, 0, C.size_t(size))), size)
	first, last := uint64(math.MaxUint64), uint64(0)
	for i, name := range s.names {
		begin, end := binary.LittleEndian.Uint64(raw[16*i:]), binary.LittleEndian.Uint64(raw[16*i+8:])
		// An empty pass leaves its end unwritten (Metal), so 0.
		if end >= begin && end != 0 {
			done(name, time.Duration(float64(end-begin)*t.period))
			first, last = min(first, begin), max(last, end)
		}
	}
	// A tiled GPU starts a pass before the one before it ends, so
	// the passes' spans overlap: frame is the whole, first stamp to
	// last.
	if last > first && len(s.names) > 1 {
		done(FrameSpan, time.Duration(float64(last-first)*t.period))
	}
	C.e_wgpuBufferUnmap(s.read.h)
}

// Span is where the pass called name writes its timestamps this frame; the
// zero Span when timers are off, the device has none, a slot still waits
// or the frame has timed timerSpans passes.
func (t *Timers) Span(name string) Span {
	s := &t.slots[t.slot]
	if !t.On || !t.d.timestamps || s.pending || len(s.names) == timerSpans {
		return Span{}
	}
	if s.set == nil {
		var a arena
		desc := alloc[C.WGPUQuerySetDescriptor](&a)
		desc._type = C.WGPUQueryType_Timestamp
		desc.count = 2 * timerSpans
		s.set = C.e_wgpuDeviceCreateQuerySet(t.d.h, desc)
		a.free()
		s.resolve = t.d.CreateBuffer(16*timerSpans, BufferQueryResolve|BufferCopySrc)
		s.read = t.d.CreateBuffer(16*timerSpans, BufferMapRead|BufferCopyDst)
		s.wait = newWait()
	}
	s.names = append(s.names, name)
	return Span{set: s.set, begin: 2 * (len(s.names) - 1)}
}

// Resolve has e copy this frame's stamps where Submitted reads them from.
func (t *Timers) Resolve(e *Encoder) {
	s := &t.slots[t.slot]
	if len(s.names) == 0 || s.pending {
		return
	}
	n := 2 * len(s.names)
	C.e_wgpuCommandEncoderResolveQuerySet(e.h, s.set, 0, C.uint32_t(n), s.resolve.h, 0)
	C.e_wgpuCommandEncoderCopyBufferToBuffer(e.h, s.resolve.h, 0, s.read.h, 0, C.uint64_t(8*n))
}

// Submitted starts reading this frame's stamps back, once its commands
// are submitted; Frame takes them timerFrames frames later.
func (t *Timers) Submitted() {
	s := &t.slots[t.slot]
	if len(s.names) == 0 || s.pending {
		return
	}
	*s.wait = C.ember_wait{}
	C.ember_map_read(s.read.h, C.size_t(16*len(s.names)), s.wait)
	s.pending = true
}

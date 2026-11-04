package wgpu

/*
#include <stdlib.h>
#include "procs.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// SurfaceSource is a native window's handle a surface draws into, per
// platform: a CAMetalLayer, an HWND, an X11 window or a Wayland surface.
type SurfaceSource interface {
	chain(a *arena) *C.WGPUChainedStruct
}

// MetalLayer is a CAMetalLayer the window's view shows (macOS, iOS).
type MetalLayer struct{ Layer unsafe.Pointer }

func (m MetalLayer) chain(a *arena) *C.WGPUChainedStruct {
	s := alloc[C.WGPUSurfaceSourceMetalLayer](a)
	s.chain.sType = C.WGPUSType_SurfaceSourceMetalLayer
	s.layer = m.Layer
	return &s.chain
}

// WindowsHWND is a Win32 window and its module (Windows).
type WindowsHWND struct{ Instance, HWND unsafe.Pointer }

func (w WindowsHWND) chain(a *arena) *C.WGPUChainedStruct {
	s := alloc[C.WGPUSurfaceSourceWindowsHWND](a)
	s.chain.sType = C.WGPUSType_SurfaceSourceWindowsHWND
	s.hinstance, s.hwnd = w.Instance, w.HWND
	return &s.chain
}

// XlibWindow is an X11 window on its display (Linux).
type XlibWindow struct {
	Display unsafe.Pointer
	Window  uint64
}

func (x XlibWindow) chain(a *arena) *C.WGPUChainedStruct {
	s := alloc[C.WGPUSurfaceSourceXlibWindow](a)
	s.chain.sType = C.WGPUSType_SurfaceSourceXlibWindow
	s.display, s.window = x.Display, C.uint64_t(x.Window)
	return &s.chain
}

// WaylandSurface is a Wayland surface on its display (Linux).
type WaylandSurface struct{ Display, Surface unsafe.Pointer }

func (w WaylandSurface) chain(a *arena) *C.WGPUChainedStruct {
	s := alloc[C.WGPUSurfaceSourceWaylandSurface](a)
	s.chain.sType = C.WGPUSType_SurfaceSourceWaylandSurface
	s.display, s.surface = w.Display, w.Surface
	return &s.chain
}

// Surface is what a window shows: a chain of textures presented in turn.
type Surface struct {
	h      C.WGPUSurface
	format TextureFormat
	width  int
	height int
}

// Surface makes the surface of a native window.
func (i *Instance) Surface(src SurfaceSource) (*Surface, error) {
	var a arena
	defer a.free()
	desc := alloc[C.WGPUSurfaceDescriptor](&a)
	desc.nextInChain = src.chain(&a)
	h := C.e_wgpuInstanceCreateSurface(i.h, desc)
	if h == nil {
		return nil, errors.New("wgpu: no surface for the window")
	}
	return &Surface{h: h}, nil
}

// Formats are the texture formats the surface can show through ad.
func (s *Surface) Formats(ad *Adapter) []TextureFormat {
	var a arena
	defer a.free()
	caps := alloc[C.WGPUSurfaceCapabilities](&a)
	if C.e_wgpuSurfaceGetCapabilities(s.h, ad.h, caps) != C.WGPUStatus_Success {
		return nil
	}
	defer C.e_wgpuSurfaceCapabilitiesFreeMembers(*caps)
	var out []TextureFormat
	for _, f := range unsafe.Slice(caps.formats, caps.formatCount) {
		out = append(out, TextureFormat(f))
	}
	return out
}

// Configure sizes the surface's textures, width by height pixels in
// format, presented at the display's rate when vsync, else at once. Its
// textures are only drawn into and shown, which lets the compositor take
// them as they are; a frame to read back draws into a texture of its own.
func (s *Surface) Configure(d *Device, width, height int, format TextureFormat, vsync bool) {
	var a arena
	defer a.free()
	c := alloc[C.WGPUSurfaceConfiguration](&a)
	c.device = d.h
	c.format = C.WGPUTextureFormat(format)
	c.usage = C.WGPUTextureUsage(UsageRenderAttachment)
	c.width, c.height = C.uint32_t(width), C.uint32_t(height)
	c.alphaMode = C.WGPUCompositeAlphaMode_Opaque
	c.presentMode = C.WGPUPresentMode_Immediate
	if vsync {
		c.presentMode = C.WGPUPresentMode_Fifo
	}
	C.e_wgpuSurfaceConfigure(s.h, c)
	s.format, s.width, s.height = format, width, height
}

// Size is the size the surface was last configured to, in pixels.
func (s *Surface) Size() (int, int) { return s.width, s.height }

// Current is the texture to draw this frame into, which Present shows.
// It is nil, with no error, when the surface is out of date and must be
// configured again.
func (s *Surface) Current() (*Texture, error) {
	var a arena
	defer a.free()
	st := alloc[C.WGPUSurfaceTexture](&a)
	C.e_wgpuSurfaceGetCurrentTexture(s.h, st)
	switch C.int(st.status) {
	case C.WGPUSurfaceGetCurrentTextureStatus_SuccessOptimal, C.WGPUSurfaceGetCurrentTextureStatus_SuccessSuboptimal:
		return &Texture{h: st.texture, Width: s.width, Height: s.height, Format: s.format, surface: true}, nil
	case C.int(C.WGPUSurfaceGetCurrentTextureStatus_Occluded):
		return nil, ErrOccluded
	case C.WGPUSurfaceGetCurrentTextureStatus_Outdated,
		C.WGPUSurfaceGetCurrentTextureStatus_Lost,
		C.WGPUSurfaceGetCurrentTextureStatus_Timeout:
		if st.texture != nil {
			C.e_wgpuTextureRelease(st.texture)
		}
		return nil, nil //nolint:nilnil // no texture this frame: the surface needs configuring, as Current's doc says
	}
	return nil, fmt.Errorf("wgpu: no surface texture (status %d)", st.status)
}

// ErrOccluded is Current's answer while the window can't be seen, so a
// frame is skipped rather than waited for.
var ErrOccluded = errors.New("wgpu: the window is occluded")

// Present shows what was drawn into Current.
func (s *Surface) Present() { C.e_wgpuSurfacePresent(s.h) }

// Release lets the surface go.
func (s *Surface) Release() {
	C.e_wgpuSurfaceUnconfigure(s.h)
	C.e_wgpuSurfaceRelease(s.h)
}

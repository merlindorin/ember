package wgpu

/*
#include <stdlib.h>
#include "procs.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// TextureFormat is how a texture's texels are stored.
type TextureFormat uint32

const (
	BGRA8Unorm TextureFormat = C.WGPUTextureFormat_BGRA8Unorm
	RGBA8Unorm TextureFormat = C.WGPUTextureFormat_RGBA8Unorm
	// RGBA8UnormSrgb decodes to linear as it is sampled.
	RGBA8UnormSrgb TextureFormat = C.WGPUTextureFormat_RGBA8UnormSrgb
	RGBA16Float    TextureFormat = C.WGPUTextureFormat_RGBA16Float
	RGBA32Float    TextureFormat = C.WGPUTextureFormat_RGBA32Float
	R32Float       TextureFormat = C.WGPUTextureFormat_R32Float
	Depth32Float   TextureFormat = C.WGPUTextureFormat_Depth32Float
	Depth24Plus    TextureFormat = C.WGPUTextureFormat_Depth24Plus
	R8Unorm        TextureFormat = C.WGPUTextureFormat_R8Unorm
)

// String is the format's WebGPU name.
func (f TextureFormat) String() string {
	switch f {
	case BGRA8Unorm:
		return "bgra8unorm"
	case RGBA8Unorm:
		return "rgba8unorm"
	case RGBA8UnormSrgb:
		return "rgba8unorm-srgb"
	case RGBA16Float:
		return "rgba16float"
	case RGBA32Float:
		return "rgba32float"
	case R32Float:
		return "r32float"
	case Depth32Float:
		return "depth32float"
	case Depth24Plus:
		return "depth24plus"
	case R8Unorm:
		return "r8unorm"
	}
	return fmt.Sprintf("format %d", uint32(f))
}

// Usage says what a texture or buffer is used for, as flags. The header
// declares flags as variables, so these are too.
type Usage uint64

// Texture is an image on the GPU.
type Texture struct {
	h             C.WGPUTexture
	Width, Height int
	Format        TextureFormat
	Samples       int
	// surface marks a surface's texture, which Present takes back.
	surface bool
	view    C.WGPUTextureView
}

// CreateTexture makes a texture of width by height texels.
func (d *Device) CreateTexture(width, height int, format TextureFormat, usage Usage) *Texture {
	return d.CreateTextureSpec(TextureSpec{Width: width, Height: height, Format: format, Usage: usage})
}

// TextureSpec is a texture's size, format and use, its samples per texel
// (1 when 0, more for multisampling) and its mip levels (1 when 0).
type TextureSpec struct {
	Width, Height int
	Format        TextureFormat
	Usage         Usage
	Samples, Mips int
}

// CreateTextureSpec makes the texture s describes.
func (d *Device) CreateTextureSpec(s TextureSpec) *Texture {
	var a arena
	defer a.free()
	desc := alloc[C.WGPUTextureDescriptor](&a)
	desc.usage = C.WGPUTextureUsage(s.Usage)
	desc.dimension = C.WGPUTextureDimension_2D
	desc.size = C.WGPUExtent3D{width: C.uint32_t(s.Width), height: C.uint32_t(s.Height), depthOrArrayLayers: 1}
	desc.format = C.WGPUTextureFormat(s.Format)
	desc.mipLevelCount, desc.sampleCount = C.uint32_t(max(s.Mips, 1)), C.uint32_t(max(s.Samples, 1))
	return &Texture{
		h:       C.e_wgpuDeviceCreateTexture(d.h, desc),
		Width:   s.Width,
		Height:  s.Height,
		Format:  s.Format,
		Samples: max(s.Samples, 1),
	}
}

// CreateTextureChecked makes the texture s describes, or says why the
// device refused it, naming its size, format and use.
func (d *Device) CreateTextureChecked(s TextureSpec) (*Texture, error) {
	var t *Texture
	if err := d.scoped(func() { t = d.CreateTextureSpec(s) }); err != nil {
		t.Release()
		return nil, fmt.Errorf(
			"wgpu: a %d×%d %s texture (usage %#x): %w",
			s.Width,
			s.Height,
			s.Format,
			uint64(s.Usage),
			err,
		)
	}
	return t, nil
}

// View is the texture's whole view, made the first time.
func (t *Texture) View() TextureView {
	if t.view == nil {
		t.view = C.e_wgpuTextureCreateView(t.h, nil)
	}
	return TextureView{h: t.view}
}

// Release lets the texture and its view go; a texture that isn't a
// surface's is destroyed.
func (t *Texture) Release() {
	if t.view != nil {
		C.e_wgpuTextureViewRelease(t.view)
		t.view = nil
	}
	if !t.surface {
		C.e_wgpuTextureDestroy(t.h)
	}
	C.e_wgpuTextureRelease(t.h)
}

// TextureView is a texture as a pass or a shader sees it.
type TextureView struct{ h C.WGPUTextureView }

// WriteTexture fills texture t from pix, rows of bytesPerRow bytes.
func (d *Device) WriteTexture(t *Texture, pix []byte, bytesPerRow int) {
	d.WriteTextureLevel(t, 0, t.Width, t.Height, pix, bytesPerRow)
}

// WriteTextureLevel fills mip level of t, width by height texels, from
// pix, rows of bytesPerRow bytes.
func (d *Device) WriteTextureLevel(t *Texture, level, width, height int, pix []byte, bytesPerRow int) {
	if len(pix) == 0 {
		return
	}
	var a arena
	defer a.free()
	dst := alloc[C.WGPUTexelCopyTextureInfo](&a)
	dst.texture = t.h
	dst.mipLevel = C.uint32_t(level)
	dst.aspect = C.WGPUTextureAspect_All
	layout := alloc[C.WGPUTexelCopyBufferLayout](&a)
	layout.bytesPerRow = C.uint32_t(bytesPerRow)
	layout.rowsPerImage = C.uint32_t(height)
	size := alloc[C.WGPUExtent3D](&a)
	size.width, size.height, size.depthOrArrayLayers = C.uint32_t(width), C.uint32_t(height), 1
	data := C.CBytes(pix)
	defer C.free(data)
	C.e_wgpuQueueWriteTexture(d.q, dst, data, C.size_t(len(pix)), layout, size)
}

// WriteTextureRegion fills the width by height texels of t at x, y from
// pix, rows of bytesPerRow bytes.
func (d *Device) WriteTextureRegion(t *Texture, x, y, width, height int, pix []byte, bytesPerRow int) {
	if len(pix) == 0 || width <= 0 || height <= 0 {
		return
	}
	var a arena
	defer a.free()
	dst := alloc[C.WGPUTexelCopyTextureInfo](&a)
	dst.texture = t.h
	dst.origin.x, dst.origin.y = C.uint32_t(x), C.uint32_t(y)
	dst.aspect = C.WGPUTextureAspect_All
	layout := alloc[C.WGPUTexelCopyBufferLayout](&a)
	layout.bytesPerRow = C.uint32_t(bytesPerRow)
	layout.rowsPerImage = C.uint32_t(height)
	size := alloc[C.WGPUExtent3D](&a)
	size.width, size.height, size.depthOrArrayLayers = C.uint32_t(width), C.uint32_t(height), 1
	data := C.CBytes(pix)
	defer C.free(data)
	C.e_wgpuQueueWriteTexture(d.q, dst, data, C.size_t(len(pix)), layout, size)
}

// Buffer is memory on the GPU.
type Buffer struct {
	h    C.WGPUBuffer
	Size int
}

// CreateBuffer makes a buffer of size bytes.
func (d *Device) CreateBuffer(size int, usage Usage) *Buffer {
	var a arena
	defer a.free()
	desc := alloc[C.WGPUBufferDescriptor](&a)
	desc.usage = C.WGPUBufferUsage(usage)
	desc.size = C.uint64_t(size)
	return &Buffer{h: C.e_wgpuDeviceCreateBuffer(d.h, desc), Size: size}
}

// CreateBufferChecked makes a buffer of size bytes, or says why the device
// refused it, naming its size and use.
func (d *Device) CreateBufferChecked(size int, usage Usage) (*Buffer, error) {
	var b *Buffer
	if err := d.scoped(func() { b = d.CreateBuffer(size, usage) }); err != nil {
		b.Release()
		return nil, fmt.Errorf("wgpu: a %d-byte buffer (usage %#x): %w", size, uint64(usage), err)
	}
	return b, nil
}

// WriteBuffer copies bytes bytes from data into b at offset, through the
// queue. Go memory is fine: the queue copies it before returning.
func (d *Device) WriteBuffer(b *Buffer, offset int, data unsafe.Pointer, bytes int) {
	if bytes == 0 {
		return
	}
	C.e_wgpuQueueWriteBuffer(d.q, b.h, C.uint64_t(offset), data, C.size_t(bytes))
}

// Release destroys the buffer.
func (b *Buffer) Release() {
	C.e_wgpuBufferDestroy(b.h)
	C.e_wgpuBufferRelease(b.h)
}

// Sampler is how a shader reads a texture.
type Sampler struct{ h C.WGPUSampler }

// CreateSampler makes a sampler: nearest or linear, clamped or repeating.
func (d *Device) CreateSampler(nearest, repeat bool) *Sampler {
	return d.CreateSamplerSpec(SamplerSpec{Nearest: nearest, Repeat: repeat})
}

// SamplerSpec is how a sampler reads: nearest or linear, clamped or
// repeating, blending mip levels, or comparing depth (less or equal) for
// shadows.
type SamplerSpec struct {
	Nearest, Repeat, Mipmaps, Compare bool
}

// CreateSamplerSpec makes the sampler s describes.
func (d *Device) CreateSamplerSpec(s SamplerSpec) *Sampler {
	var a arena
	defer a.free()
	desc := alloc[C.WGPUSamplerDescriptor](&a)
	mode := C.WGPUAddressMode(C.WGPUAddressMode_ClampToEdge)
	if s.Repeat {
		mode = C.WGPUAddressMode_Repeat
	}
	desc.addressModeU, desc.addressModeV, desc.addressModeW = mode, mode, mode
	filter := C.WGPUFilterMode(C.WGPUFilterMode_Linear)
	if s.Nearest {
		filter = C.WGPUFilterMode_Nearest
	}
	desc.magFilter, desc.minFilter = filter, filter
	desc.mipmapFilter = C.WGPUMipmapFilterMode_Nearest
	if s.Mipmaps {
		desc.mipmapFilter = C.WGPUMipmapFilterMode_Linear
	}
	if s.Compare {
		desc.compare = C.WGPUCompareFunction_LessEqual
	}
	desc.lodMinClamp, desc.lodMaxClamp = 0, 32
	desc.maxAnisotropy = 1
	return &Sampler{h: C.e_wgpuDeviceCreateSampler(d.h, desc)}
}

// Shader is a WGSL module.
type Shader struct{ h C.WGPUShaderModule }

// CreateShader compiles WGSL, and gives its compile errors as an error.
func (d *Device) CreateShader(label, wgsl string) (*Shader, error) {
	var a arena
	defer a.free()
	src := alloc[C.WGPUShaderSourceWGSL](&a)
	src.chain.sType = C.WGPUSType_ShaderSourceWGSL
	src.code = a.str(wgsl)
	desc := alloc[C.WGPUShaderModuleDescriptor](&a)
	desc.nextInChain = &src.chain
	desc.label = a.str(label)
	var h C.WGPUShaderModule
	err := d.scoped(func() { h = C.e_wgpuDeviceCreateShaderModule(d.h, desc) })
	if err != nil {
		if h != nil {
			C.e_wgpuShaderModuleRelease(h)
		}
		return nil, err
	}
	return &Shader{h: h}, nil
}

func (s *Shader) Release() { C.e_wgpuShaderModuleRelease(s.h) }

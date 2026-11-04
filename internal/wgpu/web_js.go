//go:build js

// The browser's WebGPU (navigator.gpu) behind the same API as the cgo
// binding over wgpu-native, so gpu/ draws in a page unchanged. Handles are
// JS objects. Requests that are promises in the browser (the adapter, the
// device, an error scope's error) are awaited, which blocks the calling
// goroutine until the browser settles them: only ever from a goroutine,
// never inside a js.Func callback, where blocking deadlocks.
package wgpu

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"time"
	"unsafe"

	"syscall/js"
)

const (
	LibraryEnv  = "EMBER_WGPU_LIB"
	FallbackEnv = "EMBER_WGPU_FALLBACK"
	OptionalEnv = "EMBER_GPU_OPTIONAL"
)

// await blocks until p settles: its value, or its rejection as an error.
func await(p js.Value) (js.Value, error) {
	type result struct {
		v   js.Value
		err error
	}
	done := make(chan result, 1)
	var then, catch js.Func
	then = js.FuncOf(func(_ js.Value, args []js.Value) any {
		v := js.Undefined()
		if len(args) > 0 {
			v = args[0]
		}
		done <- result{v: v}
		return nil
	})
	catch = js.FuncOf(func(_ js.Value, args []js.Value) any {
		msg := "rejected"
		if len(args) > 0 {
			msg = args[0].Call("toString").String()
		}
		done <- result{err: errors.New(msg)}
		return nil
	})
	defer then.Release()
	defer catch.Release()
	p.Call("then", then, catch)
	r := <-done
	return r.v, r.err
}

func object() js.Value { return js.Global().Get("Object").New() }

func array(vs ...any) js.Value { return js.ValueOf(vs) }

// Load has nothing to open in a browser.
func Load() error {
	if gpu().IsUndefined() {
		return errors.New("wgpu: this browser has no WebGPU (navigator.gpu)")
	}
	return nil
}

func gpu() js.Value { return js.Global().Get("navigator").Get("gpu") }

func Library() string { return "navigator.gpu" }

func Version() string { return "browser" }

func Backends() []string { return []string{"WebGPU"} }

type Instance struct{ g js.Value }

func NewInstance() (*Instance, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	return &Instance{g: gpu()}, nil
}

type Adapter struct{ v js.Value }

// RequestAdapter awaits the browser's high-performance adapter; compatible
// is ignored, as any adapter draws into any canvas.
func (i *Instance) RequestAdapter(_ *Surface) (*Adapter, error) {
	opts := object()
	opts.Set("powerPreference", "high-performance")
	v, err := await(i.g.Call("requestAdapter", opts))
	if err != nil {
		return nil, fmt.Errorf("wgpu: no adapter: %w", err)
	}
	if v.IsNull() || v.IsUndefined() {
		return nil, errors.New("wgpu: no adapter")
	}
	return &Adapter{v: v}, nil
}

type AdapterInfo struct {
	Vendor, Architecture, Device, Description, Backend string
}

func (ad *Adapter) Info() AdapterInfo {
	info := ad.v.Get("info")
	str := func(k string) string {
		if info.IsUndefined() || info.Get(k).IsUndefined() {
			return ""
		}
		return info.Get(k).String()
	}
	return AdapterInfo{
		Vendor:       str("vendor"),
		Architecture: str("architecture"),
		Device:       str("device"),
		Description:  str("description"),
		Backend:      "WebGPU",
	}
}

type Device struct {
	v, q js.Value
	// staging is one reusable Uint8Array the writes copy Go bytes into.
	staging js.Value
	// lost is the last error no error scope caught, for TakeError.
	onError js.Func
}

// lastError is the last error no error scope caught, for TakeError.
//
//nolint:gochecknoglobals // the page's one device reports it from a browser callback
var lastError string

// RequestDevice awaits the adapter's device, which reports the errors no
// scope catches through TakeError.
func (ad *Adapter) RequestDevice() (*Device, error) {
	v, err := await(ad.v.Call("requestDevice"))
	if err != nil {
		return nil, fmt.Errorf("wgpu: no device: %w", err)
	}
	d := &Device{v: v, q: v.Get("queue")}
	d.onError = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			lastError = args[0].Get("error").Get("message").String()
			js.Global().Get("console").Call("error", "wgpu:", lastError)
		}
		return nil
	})
	v.Call("addEventListener", "uncapturederror", d.onError)
	return d, nil
}

func TakeError() error {
	if lastError == "" {
		return nil
	}
	e := errors.New("wgpu: " + lastError)
	lastError = ""
	return e
}

// framing is whether a frame is being drawn: awaiting then would hand the
// browser a half-drawn canvas, so errors arrive through TakeError instead.
//
//nolint:gochecknoglobals // set by the runner around each frame
var framing bool

// Framing tells the binding a runner is drawing a frame (true) or between
// frames (false).
func Framing(on bool) { framing = on }

// scoped runs create inside a validation error scope and awaits its error;
// inside a frame it doesn't wait, and the error, if any, comes a moment
// later through TakeError.
func (d *Device) scoped(create func()) error {
	d.v.Call("pushErrorScope", "validation")
	create()
	if framing {
		var later js.Func
		later = js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) > 0 && !args[0].IsNull() && !args[0].IsUndefined() {
				lastError = args[0].Get("message").String()
			}
			later.Release()
			return nil
		})
		d.v.Call("popErrorScope").Call("then", later)
		return nil
	}
	v, err := await(d.v.Call("popErrorScope"))
	if err != nil {
		return err
	}
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	return errors.New(v.Get("message").String())
}

func (d *Device) Poll() {}

// bytesOf copies n bytes at p into the device's staging array and gives
// the part of it that holds them.
func (d *Device) bytesOf(p unsafe.Pointer, n int) js.Value {
	if d.staging.IsUndefined() || d.staging.Length() < n {
		grown := 64 << 10
		if !d.staging.IsUndefined() {
			grown = 2 * d.staging.Length()
		}
		d.staging = js.Global().Get("Uint8Array").New(max(n, grown))
	}
	js.CopyBytesToJS(d.staging, unsafe.Slice((*byte)(p), n))
	return d.staging.Call("subarray", 0, n)
}

// TextureFormat is how a texture's texels are stored.
type TextureFormat uint32

const (
	BGRA8Unorm TextureFormat = iota + 1
	RGBA8Unorm
	RGBA8UnormSrgb
	RGBA16Float
	RGBA32Float
	R32Float
	Depth32Float
	Depth24Plus
	R8Unorm
)

//nolint:gochecknoglobals // a lookup table
var formatNames = map[TextureFormat]string{
	BGRA8Unorm:     "bgra8unorm",
	RGBA8Unorm:     "rgba8unorm",
	RGBA8UnormSrgb: "rgba8unorm-srgb",
	RGBA16Float:    "rgba16float",
	RGBA32Float:    "rgba32float",
	R32Float:       "r32float",
	Depth32Float:   "depth32float",
	Depth24Plus:    "depth24plus",
	R8Unorm:        "r8unorm",
}

func (f TextureFormat) String() string {
	if n, ok := formatNames[f]; ok {
		return n
	}
	return fmt.Sprintf("format %d", uint32(f))
}

// FormatNamed is the format WebGPU names name, 0 for none ours.
func FormatNamed(name string) TextureFormat {
	for f, n := range formatNames {
		if n == name {
			return f
		}
	}
	return 0
}

type Usage uint64

// The flags are WebGPU's own numbers, as webgpu.h's.
const (
	UsageCopySrc          Usage = 1
	UsageCopyDst          Usage = 2
	UsageTextureBinding   Usage = 4
	UsageRenderAttachment Usage = 16

	BufferMapRead      Usage = 1
	BufferMapWrite     Usage = 2
	BufferCopySrc      Usage = 4
	BufferCopyDst      Usage = 8
	BufferIndex        Usage = 16
	BufferVertex       Usage = 32
	BufferUniform      Usage = 64
	BufferQueryResolve Usage = 512
)

type Texture struct {
	v             js.Value
	Width, Height int
	Format        TextureFormat
	Samples       int
	surface       bool
	view          *jsView
}

func (d *Device) CreateTexture(width, height int, format TextureFormat, usage Usage) *Texture {
	return d.CreateTextureSpec(TextureSpec{Width: width, Height: height, Format: format, Usage: usage})
}

type TextureSpec struct {
	Width, Height int
	Format        TextureFormat
	Usage         Usage
	Samples, Mips int
}

func (d *Device) CreateTextureSpec(s TextureSpec) *Texture {
	desc := object()
	desc.Set("size", array(s.Width, s.Height, 1))
	desc.Set("format", s.Format.String())
	desc.Set("usage", uint64(s.Usage))
	desc.Set("mipLevelCount", max(s.Mips, 1))
	desc.Set("sampleCount", max(s.Samples, 1))
	return &Texture{
		v:       d.v.Call("createTexture", desc),
		Width:   s.Width,
		Height:  s.Height,
		Format:  s.Format,
		Samples: max(s.Samples, 1),
	}
}

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

func (t *Texture) View() TextureView {
	if t.view == nil {
		t.view = &jsView{t.v.Call("createView")}
	}
	return TextureView{t.view}
}

func (t *Texture) Release() {
	t.view = nil
	if !t.surface {
		t.v.Call("destroy")
	}
}

// TextureView is a texture as a pass or a shader sees it: comparable, as
// the renderer keys bind groups by view, so it holds the JS view through
// a pointer, nil for none.
type TextureView struct{ p *jsView }

type jsView struct{ v js.Value }

func (t TextureView) js() js.Value {
	if t.p == nil {
		return js.Undefined()
	}
	return t.p.v
}

func (d *Device) WriteTexture(t *Texture, pix []byte, bytesPerRow int) {
	d.WriteTextureLevel(t, 0, t.Width, t.Height, pix, bytesPerRow)
}

func (d *Device) WriteTextureLevel(t *Texture, level, width, height int, pix []byte, bytesPerRow int) {
	d.writeTexture(t, level, 0, 0, width, height, pix, bytesPerRow)
}

func (d *Device) WriteTextureRegion(t *Texture, x, y, width, height int, pix []byte, bytesPerRow int) {
	d.writeTexture(t, 0, x, y, width, height, pix, bytesPerRow)
}

func (d *Device) writeTexture(t *Texture, level, x, y, width, height int, pix []byte, bytesPerRow int) {
	if len(pix) == 0 || width <= 0 || height <= 0 {
		return
	}
	dst := object()
	dst.Set("texture", t.v)
	dst.Set("mipLevel", level)
	dst.Set("origin", array(x, y))
	layout := object()
	layout.Set("bytesPerRow", bytesPerRow)
	layout.Set("rowsPerImage", height)
	d.q.Call("writeTexture", dst, d.bytesOf(unsafe.Pointer(&pix[0]), len(pix)), layout, array(width, height, 1))
}

type Buffer struct {
	v    js.Value
	Size int
}

func (d *Device) CreateBuffer(size int, usage Usage) *Buffer {
	desc := object()
	desc.Set("size", size)
	desc.Set("usage", uint64(usage))
	return &Buffer{v: d.v.Call("createBuffer", desc), Size: size}
}

func (d *Device) CreateBufferChecked(size int, usage Usage) (*Buffer, error) {
	var b *Buffer
	if err := d.scoped(func() { b = d.CreateBuffer(size, usage) }); err != nil {
		b.Release()
		return nil, fmt.Errorf("wgpu: a %d-byte buffer (usage %#x): %w", size, uint64(usage), err)
	}
	return b, nil
}

func (d *Device) WriteBuffer(b *Buffer, offset int, data unsafe.Pointer, bytes int) {
	if bytes == 0 {
		return
	}
	d.q.Call("writeBuffer", b.v, offset, d.bytesOf(data, bytes))
}

func (b *Buffer) Release() { b.v.Call("destroy") }

type Sampler struct{ v js.Value }

func (d *Device) CreateSampler(nearest, repeat bool) *Sampler {
	return d.CreateSamplerSpec(SamplerSpec{Nearest: nearest, Repeat: repeat})
}

type SamplerSpec struct {
	Nearest, Repeat, Mipmaps, Compare bool
}

func (d *Device) CreateSamplerSpec(s SamplerSpec) *Sampler {
	desc := object()
	mode := "clamp-to-edge"
	if s.Repeat {
		mode = "repeat"
	}
	desc.Set("addressModeU", mode)
	desc.Set("addressModeV", mode)
	desc.Set("addressModeW", mode)
	filter := "linear"
	if s.Nearest {
		filter = "nearest"
	}
	desc.Set("magFilter", filter)
	desc.Set("minFilter", filter)
	desc.Set("mipmapFilter", map[bool]string{false: "nearest", true: "linear"}[s.Mipmaps])
	if s.Compare {
		desc.Set("compare", "less-equal")
	}
	desc.Set("lodMaxClamp", 32)
	return &Sampler{v: d.v.Call("createSampler", desc)}
}

type Shader struct{ v js.Value }

// CreateShader compiles WGSL and gives its compile errors as an error,
// read from the module's compilation info.
func (d *Device) CreateShader(label, wgsl string) (*Shader, error) {
	desc := object()
	desc.Set("label", label)
	desc.Set("code", wgsl)
	var m js.Value
	if err := d.scoped(func() { m = d.v.Call("createShaderModule", desc) }); err != nil {
		info, ierr := await(m.Call("getCompilationInfo"))
		if ierr == nil {
			var lines []string
			for i := range info.Get("messages").Length() {
				msg := info.Get("messages").Index(i)
				lines = append(
					lines,
					fmt.Sprintf(
						"%d:%d: %s",
						msg.Get("lineNum").Int(),
						msg.Get("linePos").Int(),
						msg.Get("message").String(),
					),
				)
			}
			if len(lines) > 0 {
				return nil, fmt.Errorf("wgpu: shader %s: %s", label, strings.Join(lines, "\n"))
			}
		}
		return nil, fmt.Errorf("wgpu: shader %s: %w", label, err)
	}
	return &Shader{v: m}, nil
}

func (s *Shader) Release() {}

// Surface is a canvas's WebGPU context, the window a page draws into.
type Surface struct {
	ctx           js.Value
	width, height int
}

// CanvasSurface is the WebGPU context of the canvas whose id is id.
func CanvasSurface(id string) (*Surface, error) {
	c := js.Global().Get("document").Call("getElementById", id)
	if c.IsNull() {
		return nil, fmt.Errorf("wgpu: no canvas #%s", id)
	}
	ctx := c.Call("getContext", "webgpu")
	if ctx.IsNull() {
		return nil, errors.New("wgpu: the canvas has no webgpu context")
	}
	return &Surface{ctx: ctx, width: c.Get("width").Int(), height: c.Get("height").Int()}, nil
}

// PreferredFormat is the format the browser draws canvases in fastest.
func PreferredFormat() TextureFormat {
	return FormatNamed(gpu().Call("getPreferredCanvasFormat").String())
}

func (s *Surface) Configure(d *Device, width, height int, format TextureFormat, _ bool) {
	cfg := object()
	cfg.Set("device", d.v)
	cfg.Set("format", format.String())
	cfg.Set("alphaMode", "opaque")
	cfg.Set("usage", uint64(UsageRenderAttachment|UsageCopySrc))
	s.ctx.Call("configure", cfg)
	s.width, s.height = width, height
}

func (s *Surface) Size() (int, int) { return s.width, s.height }

// Current is the canvas's texture this frame; the browser shows it once
// the frame's task ends, so Present has nothing to do.
func (s *Surface) Current() (*Texture, error) {
	t := s.ctx.Call("getCurrentTexture")
	return &Texture{
		v:       t,
		Width:   t.Get("width").Int(),
		Height:  t.Get("height").Int(),
		Format:  FormatNamed(t.Get("format").String()),
		Samples: 1,
		surface: true,
	}, nil
}

var ErrOccluded = errors.New("wgpu: the window is occluded")

func (s *Surface) Present() {}

func (s *Surface) Release() { s.ctx.Call("unconfigure") }

const readbackSlots = 4

// Readback copies small textures back without waiting: mapAsync settles a
// frame or more later, between frames, and Done hands what has come back.
type Readback struct {
	d     *Device
	slots [readbackSlots]readbackSlot
	next  int
}

type readbackSlot struct {
	buf                        *Buffer
	row, height                int
	tag                        any
	mapped, pending, ready, ok bool
	settle                     js.Func
	bytes                      []byte
}

func (d *Device) NewReadback() *Readback { return &Readback{d: d} }

// copyRows records a copy of t into buf, rows of row bytes.
func copyRows(e *Encoder, t *Texture, buf *Buffer, row int) {
	src := object()
	src.Set("texture", t.v)
	dst := object()
	dst.Set("buffer", buf.v)
	dst.Set("bytesPerRow", row)
	dst.Set("rowsPerImage", t.Height)
	e.v.Call("copyTextureToBuffer", src, dst, array(t.Width, t.Height, 1))
}

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
	copyRows(e, t, s.buf, row)
	s.row, s.height, s.tag, s.mapped = row, t.Height, tag, true
	return true
}

func (r *Readback) Submitted() {
	s := &r.slots[r.next]
	if !s.mapped || s.pending {
		return
	}
	if s.settle.IsUndefined() {
		s.settle = js.FuncOf(func(_ js.Value, args []js.Value) any {
			s.ready, s.ok = true, len(args) == 0 || args[0].IsUndefined()
			return nil
		})
	}
	s.ready = false
	s.buf.v.Call("mapAsync", 1, 0, s.row*s.height).Call("then", s.settle, s.settle)
	s.pending, s.mapped = true, false
	r.next = (r.next + 1) % readbackSlots
}

func (r *Readback) Done(read func(rows []byte, row int, tag any)) {
	for k := range readbackSlots {
		s := &r.slots[(r.next+k)%readbackSlots]
		if !s.pending || !s.ready {
			continue
		}
		s.pending = false
		if !s.ok {
			continue
		}
		size := s.row * s.height
		if cap(s.bytes) < size {
			s.bytes = make([]byte, size)
		}
		s.bytes = s.bytes[:size]
		js.CopyBytesToGo(s.bytes, js.Global().Get("Uint8Array").New(s.buf.v.Call("getMappedRange", 0, size)))
		s.buf.v.Call("unmap")
		read(s.bytes, s.row, s.tag)
	}
}

// Read copies t back, waiting for it, as an opaque RGBA image, top row
// first: a capture. It awaits, so only between frames.
func (d *Device) Read(t *Texture) (*image.RGBA, error) {
	if framing {
		return nil, errors.New("wgpu: Read inside a frame")
	}
	row := (t.Width*4 + 255) &^ 255
	size := row * t.Height
	buf := d.CreateBuffer(size, BufferCopyDst|BufferMapRead)
	defer buf.Release()
	e := d.CreateEncoder()
	copyRows(e, t, buf, row)
	e.Submit()
	if _, err := await(buf.v.Call("mapAsync", 1, 0, size)); err != nil {
		return nil, fmt.Errorf("wgpu: read back: %w", err)
	}
	mapped := make([]byte, size)
	js.CopyBytesToGo(mapped, js.Global().Get("Uint8Array").New(buf.v.Call("getMappedRange", 0, size)))
	buf.v.Call("unmap")
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

const FrameSpan = "frame"

// Timers time passes where the device can; browsers mostly hide
// timestamps, so on the web no span is ever timed.
type Timers struct {
	On bool
}

type Span struct{}

func (d *Device) NewTimers() *Timers { return &Timers{} }

func (t *Timers) Frame(_ func(name string, d time.Duration)) {}

func (t *Timers) Span(_ string) Span { return Span{} }

func (t *Timers) Resolve(_ *Encoder) {}

func (t *Timers) Submitted() {}

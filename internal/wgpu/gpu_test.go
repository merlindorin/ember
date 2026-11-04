//go:build gpu

package wgpu //nolint:testpackage // white-box: reaches the binding's unexported handles

import (
	"os"
	"strings"
	"testing"
	"time"
	"unsafe"
)

// The tests here need a GPU and the wgpu-native library (task wgpu):
//
//	go test -tags gpu ./internal/wgpu

func device(t *testing.T) *Device {
	t.Helper()
	inst, err := NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	ad, err := inst.RequestAdapter(nil)
	if err != nil {
		if os.Getenv(OptionalEnv) == "1" {
			t.Skipf("no GPU adapter here, and %s=1: %v", OptionalEnv, err)
		}
		t.Fatal(err)
	}
	t.Logf("wgpu-native %s, %+v", Version(), ad.Info())
	d, err := ad.RequestDevice()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

const triangle = `
struct Out { @builtin(position) pos: vec4f, @location(0) color: vec4f };
@vertex fn vs(@location(0) p: vec2f, @location(1) c: vec4f) -> Out { return Out(vec4f(p, 0.0, 1.0), c); }
@fragment fn fs(in: Out) -> @location(0) vec4f { return in.color; }
`

func TestATriangleDrawsIntoATextureAndReadsBack(t *testing.T) {
	d := device(t)
	sh, err := d.CreateShader("triangle", triangle)
	if err != nil {
		t.Fatal(err)
	}
	type vertex struct{ x, y, r, g, b, a float32 }
	pl, err := d.CreateRenderPipeline(PipelineSpec{
		Label:    "triangle",
		Shader:   sh,
		Vertex:   "vs",
		Fragment: "fs",
		Stride: unsafe.Sizeof(
			vertex{},
		),
		Attributes: []Attribute{{0, Float32x2, 0}, {1, Float32x4, 8}},
		Format:     RGBA8Unorm,
	})
	if err != nil {
		t.Fatal(err)
	}
	verts := []vertex{{-1, -1, 1, 0, 0, 1}, {3, -1, 1, 0, 0, 1}, {-1, 3, 1, 0, 0, 1}}
	vb := d.CreateBuffer(int(unsafe.Sizeof(vertex{}))*len(verts), BufferVertex|BufferCopyDst)
	d.WriteBuffer(vb, 0, unsafe.Pointer(&verts[0]), vb.Size)
	tex := d.CreateTexture(64, 32, RGBA8Unorm, UsageRenderAttachment|UsageCopySrc)
	e := d.CreateEncoder()
	p := e.BeginPass(tex.View(), &Color{0, 0, 1, 1})
	p.SetPipeline(pl)
	p.SetVertexBuffer(vb, 0, vb.Size)
	p.SetScissor(0, 0, 32, 32)
	p.Draw(3, 0)
	p.End()
	e.Submit()
	img, err := d.Read(tex)
	if err != nil {
		t.Fatal(err)
	}
	if c := img.RGBAAt(10, 10); c.R != 255 || c.B != 0 {
		t.Fatalf("inside the scissor the triangle is red: %v", c)
	}
	if c := img.RGBAAt(50, 10); c.B != 255 || c.R != 0 {
		t.Fatalf("outside it the clear stays blue: %v", c)
	}
	if err = TakeError(); err != nil {
		t.Fatal(err)
	}
}

func TestBadWGSLIsAnErrorWithItsLine(t *testing.T) {
	d := device(t)
	_, err := d.CreateShader("bad", "@fragment fn fs() -> @location(0) vec4f {\n  return nope;\n}")
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("the compile error names what's wrong: %v", err)
	}
	t.Log(err)
}

func TestARefusedTextureOrBufferIsAnErrorNamingWhatWasAsked(t *testing.T) {
	d := device(t)
	if _, err := d.CreateTextureChecked(
		TextureSpec{Width: 100000, Height: 16, Format: R32Float, Usage: UsageRenderAttachment},
	); err == nil ||
		!strings.Contains(err.Error(), "100000×16 r32float") {
		t.Fatalf("a refused texture names its size and format: %v", err)
	} else {
		t.Log(err)
	}
	if _, err := d.CreateBufferChecked(
		1<<40,
		BufferUniform,
	); err == nil ||
		!strings.Contains(err.Error(), "1099511627776-byte") {
		t.Fatalf("a refused buffer names its size: %v", err)
	}
	if tex, err := d.CreateTextureChecked(
		TextureSpec{Width: 64, Height: 16, Format: RGBA16Float, Usage: UsageRenderAttachment},
	); err != nil ||
		tex == nil {
		t.Fatalf("a texture the device can make comes back: %v", err)
	}
}

func TestTimedPassesReportHowLongTheGPUTook(t *testing.T) {
	d := device(t)
	if !d.timestamps {
		t.Skip("this adapter has no timestamp queries")
	}
	tex := d.CreateTexture(256, 256, RGBA8Unorm, UsageRenderAttachment)
	sh, err := d.CreateShader("triangle", triangle)
	if err != nil {
		t.Fatal(err)
	}
	type vertex struct{ x, y, r, g, b, a float32 }
	pl, err := d.CreateRenderPipeline(PipelineSpec{
		Label:    "triangle",
		Shader:   sh,
		Vertex:   "vs",
		Fragment: "fs",
		Stride: unsafe.Sizeof(
			vertex{},
		),
		Attributes: []Attribute{{0, Float32x2, 0}, {1, Float32x4, 8}},
		Format:     RGBA8Unorm,
	})
	if err != nil {
		t.Fatal(err)
	}
	verts := []vertex{{-1, -1, 1, 0, 0, 1}, {3, -1, 1, 0, 0, 1}, {-1, 3, 1, 0, 0, 1}}
	vb := d.CreateBuffer(int(unsafe.Sizeof(vertex{}))*len(verts), BufferVertex|BufferCopyDst)
	d.WriteBuffer(vb, 0, unsafe.Pointer(&verts[0]), vb.Size)
	timers := d.NewTimers()
	timers.On = true
	got := map[string]time.Duration{}
	for range 3 * timerFrames {
		timers.Frame(func(name string, took time.Duration) { got[name] = took })
		e := d.CreateEncoder()
		for _, name := range []string{"triangle", "again"} {
			p := e.BeginPass(tex.View(), &Color{0, 0, 0, 1}, timers.Span(name))
			p.SetPipeline(pl)
			p.SetVertexBuffer(vb, 0, vb.Size)
			p.Draw(3, 0)
			p.End()
		}
		timers.Resolve(e)
		e.Submit()
		timers.Submitted()
		d.Poll()
	}
	if took, ok := got["triangle"]; !ok || took <= 0 || took > time.Second {
		t.Fatalf("the pass's time comes back a few frames later: %v", got)
	}
	if whole, ok := got[FrameSpan]; !ok || whole < got["triangle"] || whole < got["again"] {
		t.Fatalf("the frame's whole span, first stamp to last, holds each pass's: %v", got)
	}
	timers.On = false
	if s := timers.Span("off"); s.set != nil {
		t.Fatal("off, nothing is timed")
	}
}

func TestAReadbackHandsACopyBackAFewFramesLaterWithoutWaiting(t *testing.T) {
	d := device(t)
	tex := d.CreateTexture(4, 1, RGBA8Unorm, UsageRenderAttachment|UsageCopySrc)
	r := d.NewReadback()
	got := map[any][]byte{}
	for frame := range 8 {
		e := d.CreateEncoder()
		p := e.BeginPass(tex.View(), &Color{R: 1, G: 0.5, B: 0, A: 1})
		p.End()
		r.Copy(e, tex, 4, frame)
		e.Submit()
		r.Submitted()
		d.Poll()
		r.Done(func(rows []byte, _ int, tag any) { got[tag] = append([]byte(nil), rows[:16]...) })
	}
	if len(got) == 0 {
		t.Fatal("copies come back")
	}
	for tag, px := range got {
		if px[0] != 255 || px[1] < 126 || px[1] > 129 || px[3] != 255 {
			t.Fatalf("copy %v holds what was drawn: %v", tag, px[:4])
		}
	}
}

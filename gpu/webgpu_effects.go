package gpu

import (
	"errors"
	"unsafe"

	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/errlog"
	"github.com/merlindorin/ember/internal/wgpu"
	"github.com/merlindorin/ember/render"
)

// errNoSource is the error of an effect without a source.
var errNoSource = errors.New("has no WGSL source")

// webgpuEffect is an effect's shader, made from its WGSL, its uniforms'
// layout and which way its vUV runs, its pipelines by target format, and
// the buffer its inputs and uniforms go through.
type webgpuEffect struct {
	key       string
	shader    *wgpu.Shader
	layout    *wgpu.BindGroupLayout
	pipelines map[wgpu.TextureFormat]*wgpu.RenderPipeline
	buffer    *wgpu.Buffer
	params    []param
	size      int
	failed    bool
}

// effectParams is where an effect's own uniforms start in its buffer,
// after its inputs: WebGPU's uniform offset alignment.
const effectParams = 256

// effectScene is what an effect is run on: its source's size, its
// output's, the time and the scene's depth range.
type effectScene struct {
	size, out  [2]float32
	time       float32
	depthRange [3]float32
}

// effect is e's shader for its WGSL and its uniforms, made again when
// any changes; nil when it has no source or doesn't compile, which Err
// says.
func (r *webgpu2D) effect(e *render.Effect) *webgpuEffect {
	if e.Shader() == "" {
		e.SetErr(errNoSource)
		return nil
	}
	ps, size, sig := params(e)
	key := e.Shader() + "\x00" + sig
	if r.effects == nil {
		r.effects = map[*render.Effect]*webgpuEffect{}
	}
	fx := r.effects[e]
	if fx != nil && fx.key == key {
		fx.params = ps
		if fx.failed {
			return nil
		}
		return fx
	}
	if fx != nil {
		fx.release()
	}
	fx = &webgpuEffect{key: key, params: ps, size: size, pipelines: make(map[wgpu.TextureFormat]*wgpu.RenderPipeline)}
	r.effects[e] = fx
	err := fx.build(r.dev, effectModule(e.Shader(), ps), e.Name())
	e.SetErr(err)
	if err != nil {
		fx.failed = true
		errlog.Print("gpu: effect " + e.Name() + ": " + err.Error())
		return nil
	}
	return fx
}

func (fx *webgpuEffect) build(d *wgpu.Device, module, name string) error {
	sh, err := d.CreateShader("effect "+name, module)
	if err != nil {
		return err
	}
	fx.shader = sh
	if fx.layout, err = d.CreateBindGroupLayout(
		wgpu.LayoutEntry{Binding: 0, Visibility: wgpu.StageFragment, Kind: wgpu.BindUniform},
		wgpu.LayoutEntry{Binding: 1, Visibility: wgpu.StageFragment, Kind: wgpu.BindTexture},
		wgpu.LayoutEntry{Binding: 2, Visibility: wgpu.StageFragment, Kind: wgpu.BindSampler},
		wgpu.LayoutEntry{Binding: 3, Visibility: wgpu.StageFragment, Kind: wgpu.BindUnfilterable},
		wgpu.LayoutEntry{Binding: 4, Visibility: wgpu.StageFragment, Kind: wgpu.BindSamplerNearest},
		wgpu.LayoutEntry{Binding: 5, Visibility: wgpu.StageFragment, Kind: wgpu.BindUniform},
	); err != nil {
		return err
	}
	fx.buffer = d.CreateBuffer(effectParams+fx.size, wgpu.BufferUniform|wgpu.BufferCopyDst)
	_, err = fx.pipeline(d, passFormat, name)
	return err
}

// pipeline is fx's pipeline drawing into format, made the first time.
func (fx *webgpuEffect) pipeline(d *wgpu.Device, format wgpu.TextureFormat, name string) (*wgpu.RenderPipeline, error) {
	if p, ok := fx.pipelines[format]; ok {
		return p, nil
	}
	p, err := d.CreateRenderPipeline(
		wgpu.PipelineSpec{Label: "effect " + name, Layouts: []*wgpu.BindGroupLayout{fx.layout},
			Shader: fx.shader, Vertex: "ember_vs", Fragment: "ember_fs", Format: format},
	)
	if err != nil {
		return nil, err
	}
	fx.pipelines[format] = p
	return p, nil
}

func (fx *webgpuEffect) release() {
	for _, p := range fx.pipelines {
		p.Release()
	}
	if fx.layout != nil {
		fx.layout.Release()
	}
	if fx.shader != nil {
		fx.shader.Release()
	}
	if fx.buffer != nil {
		fx.buffer.Release()
	}
}

// runEffect draws eff from src into into, with depth (none for a 2D
// layer) and sc, and reports whether it ran.
func (r *webgpu2D) runEffect(e *wgpu.Encoder, eff *render.Effect, src, into, depth *wgpu.Texture, sc effectScene) bool {
	fx := r.effect(eff)
	if fx == nil {
		return false
	}
	pl, err := fx.pipeline(r.dev, into.Format, eff.Name())
	if err != nil {
		eff.SetErr(err)
		return false
	}
	if depth == nil {
		depth = r.farDepth
	}
	in := effectInputs{Size: sc.size, OutputSize: sc.out, DepthRange: sc.depthRange, Time: sc.time}
	r.dev.WriteBuffer(fx.buffer, 0, unsafe.Pointer(&in), int(unsafe.Sizeof(in)))
	data := make([]byte, fx.size)
	pack(data, fx.params)
	r.dev.WriteBuffer(fx.buffer, effectParams, unsafe.Pointer(&data[0]), len(data))
	g := r.dev.CreateBindGroup(fx.layout,
		wgpu.BindEntry{Binding: 0, Buffer: fx.buffer, Size: int(unsafe.Sizeof(in))},
		wgpu.BindEntry{Binding: 1, View: src.View()},
		wgpu.BindEntry{Binding: 2, Sampler: r.samplers[0][0]},
		wgpu.BindEntry{Binding: 3, View: depth.View()},
		wgpu.BindEntry{Binding: 4, Sampler: r.samplers[1][0]},
		wgpu.BindEntry{Binding: 5, Buffer: fx.buffer, Offset: effectParams, Size: fx.size})
	pass := e.BeginPass(into.View(), &wgpu.Color{})
	pass.SetPipeline(pl)
	pass.SetBindGroup(0, g)
	pass.Draw(3, 0)
	pass.End()
	g.Release()
	return true
}

// affect runs p's effects on what t holds, each from one of its textures
// into the other, leaving the result in t.gpu.
func (r *webgpu2D) affect(e *wgpu.Encoder, t *passTarget, p *draw.Pass) {
	size := [2]float32{float32(p.Width), float32(p.Height)}
	for _, eff := range p.Effects {
		if eff == nil || !eff.Enabled() {
			continue
		}
		if t.spare == nil {
			t.spare = r.dev.CreateTexture(
				p.Width,
				p.Height,
				passFormat,
				wgpu.UsageRenderAttachment|wgpu.UsageTextureBinding|wgpu.UsageCopySrc,
			)
		}
		if r.runEffect(e, eff, t.gpu, t.spare, nil, effectScene{size: size, out: size, time: p.List.Time()}) {
			t.gpu, t.spare = t.spare, t.gpu
		}
	}
}

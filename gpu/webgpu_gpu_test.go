//go:build gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"errors"
	"strings"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"

	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/wgpu"
)

// The WebGPU presenter's shader and pipelines, on a real GPU (task wgpu):
//
//	go test -tags gpu ./gpu
func TestTheUIShaderCompilesAndEveryBlendMakesAPipeline(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	for _, f := range []wgpu.TextureFormat{wgpu.BGRA8Unorm, wgpu.RGBA8Unorm} {
		for _, m := range []draw.Blend{draw.Alpha, draw.Add, draw.Multiply, draw.Screen, draw.Lighten} {
			if _, err := r.pipeline(m, target2D{format: f}); err != nil {
				t.Fatalf("blend %d into %d: %v", m, f, err)
			}
		}
	}
	if len(r.pipelines) != 10 {
		t.Fatalf("one pipeline per blend and format, cached: %d", len(r.pipelines))
	}
	if _, err := r.pipeline(draw.Add, target2D{format: wgpu.RGBA8Unorm}); err != nil || len(r.pipelines) != 10 {
		t.Fatal("asked again, it comes from the cache")
	}
}

func TestAnEffectsModuleCompilesWithEveryKindOfUniform(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	e := render.NewEffect(`
fn effect(vUV: vec2f) -> vec4f {
	let c = color(vUV) * uTint;
	return vec4f(c.rgb * uWeights[1] + uOffsets[0].xyx * uStrength, c.a) +
		vec4f(uRipples[3].w, uSize / uOutputSize, linearDepth(vUV) * uTime);
}`)
	e.SetFloat("uStrength", 0.5)
	e.SetColor("uTint", color.White)
	e.SetFloats("uWeights", []float32{1, 2})
	e.SetVec2s("uOffsets", []geom.Vec2{{X: 1}})
	e.SetVec4s("uRipples", make([]geom.Vec4, 4))
	if r.effect(e) == nil || e.Err() != nil {
		t.Fatalf("it compiles: %v", e.Err())
	}
	bad := render.NewEffect("fn effect(vUV: vec2f) -> vec4f { return nope; }")
	if r.effect(bad) != nil || bad.Err() == nil || !strings.Contains(bad.Err().Error(), "nope") {
		t.Fatalf("a broken effect says why in Err: %v", bad.Err())
	}
	empty := render.NewEffect("")
	if r.effect(empty) != nil || !errors.Is(empty.Err(), errNoSource) {
		t.Fatalf("an effect without a source says so: %v", empty.Err())
	}
}

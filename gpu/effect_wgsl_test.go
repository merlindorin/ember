package gpu //nolint:testpackage // tests the unexported assembly of effect shaders

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnEffectsUniformsAreLaidOutAsWGSLLaysAUniformStruct(t *testing.T) {
	e := render.NewEffect("fn effect(vUV: vec2f) -> vec4f { return color(vUV); }")
	e.SetFloat("a", 1)
	e.SetVec3("b", geom.V3(2, 3, 4))
	e.SetFloat("c", 5)
	e.SetVec2("d", geom.V2(6, 7))
	e.SetFloats("e", []float32{8, 9})
	e.SetColor("f", color.RGBA(0.1, 0.2, 0.3, 0.4))
	e.SetVec4s("g", []geom.Vec4{{X: 1}, {Y: 2}, {Z: 3}})
	e.SetVec2s("h", nil)
	ps, size, sig := params(e)
	got := map[string][2]int{}
	for _, p := range ps {
		got[p.name] = [2]int{p.offset, p.size}
	}
	assert.Equal(t, map[string][2]int{
		"a": {0, 4},   // f32
		"b": {16, 12}, // vec3f, 16-byte aligned
		"c": {28, 4},  // an f32 fits after a vec3f
		"d": {32, 8},  // vec2f, 8-byte aligned
		"e": {48, 32}, // array of 16-byte elements, 16-byte aligned
		"f": {80, 16}, // a colour is a vec4f
		"g": {96, 48},
	}, got, "by name; an empty slice has no member")
	assert.Equal(t, 144, size, "a multiple of 16")
	assert.Equal(t, "a:f32;b:vec3f;c:f32;d:vec2f;e:array<vec4f, 2>;f:vec4f;g:array<vec4f, 3>;", sig)

	data := make([]byte, size)
	pack(data, ps)
	f := func(at int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(data[at:])) }
	assert.Equal(t, []float32{1, 2, 3, 4, 5, 6, 7}, []float32{f(0), f(16), f(20), f(24), f(28), f(32), f(36)})
	assert.Equal(t, []float32{8, 0, 9}, []float32{f(48), f(52), f(64)}, "each float in its own 16 bytes")
	assert.Equal(t, float32(3), f(96+2*16+8))

	e.SetFloats("e", []float32{1, 2, 3})
	_, _, again := params(e)
	assert.NotEqual(t, sig, again, "a longer array changes the signature, so the module is made again")
}

func TestTheEffectModuleGivesTheEffectItsInputsUnderTheirNames(t *testing.T) {
	e := render.NewEffect("")
	e.SetFloats("uWeights", []float32{1, 2})
	e.SetVec4s("uRipples", make([]geom.Vec4, 4))
	e.SetFloat("uStrength", 1)
	ps, _, _ := params(e)
	m := effectModule("fn effect(vUV: vec2f) -> vec4f { return color(vUV); }", ps)
	for _, want := range []string{
		"uRipples: array<vec4f, 4>,", "var<private> uRipples: array<vec4f, 4>;", "uRipples = emberParams.uRipples;",
		"uWeights: array<vec4f, 2>,", "var<private> uWeights: array<f32, 2>;", "uWeights[i] = emberParams.uWeights[i].x;",
		"var<private> uStrength: f32;", "uStrength = emberParams.uStrength;",
		"uSize = emberInputs.uSize;", "uTime = emberInputs.uTime;", "fn linearDepth(uv: vec2f) -> f32",
		"return effect(in.uv);",
	} {
		assert.Contains(t, m, want)
	}
	require.True(t, strings.HasSuffix(m, "fn effect(vUV: vec2f) -> vec4f { return color(vUV); }"), "the source last")
}

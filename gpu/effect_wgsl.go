package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/render"
)

// param is one of an effect's own uniforms in its Params struct, laid out
// as WGSL lays a uniform struct out: f32 4 bytes, vec2f 8, vec3f and
// vec4f 16-byte aligned, and arrays of 16-byte elements whatever they
// hold. Each element of an array is copied into a private array of its
// own type, so the effect reads uWeights[i] as an f32.
type param struct {
	name   string
	stored string // the type in the Params struct
	seen   string // the type the effect reads
	offset int
	size   int
	// count is the array's length, 0 for a single value; swizzle picks
	// what of each 16-byte element the effect sees.
	count   int
	swizzle string
	values  []float32
}

// params lays out e's uniforms, by name: their struct's members, its size
// in bytes (16 at least, a multiple of 16) and a signature that changes
// whenever a member's type or length does.
func params(e *render.Effect) ([]param, int, string) {
	var names []string
	for name := range e.Uniforms() {
		names = append(names, name)
	}
	slices.Sort(names)
	var out []param
	at := 0
	for _, name := range names {
		v, _ := e.Uniform(name)
		p, ok := paramOf(name, v)
		if !ok {
			continue
		}
		align := 4
		switch {
		case p.count > 0, p.stored == "vec3f", p.stored == "vec4f":
			align = 16
		case p.stored == "vec2f":
			align = 8
		}
		at = (at + align - 1) / align * align
		p.offset = at
		at += p.size
		out = append(out, p)
	}
	size := max((at+15)/16*16, 16)
	var sig strings.Builder
	for _, p := range out {
		fmt.Fprintf(&sig, "%s:%s;", p.name, p.stored)
	}
	return out, size, sig.String()
}

// paramOf is the member a uniform of value v is: single floats, vectors
// and colours as themselves, slices as arrays of 16-byte elements. Empty
// slices and other types have none.
func paramOf(name string, v any) (param, bool) {
	one := func(stored string, size int, values ...float32) (param, bool) {
		return param{name: name, stored: stored, seen: stored, size: size, values: values}, true
	}
	many := func(seen, swizzle string, n, per int, at func(i int) []float32) (param, bool) {
		if n == 0 {
			return param{}, false
		}
		p := param{name: name, stored: fmt.Sprintf("array<vec4f, %d>", n), seen: fmt.Sprintf("array<%s, %d>", seen, n),
			size: 16 * n, count: n, swizzle: swizzle}
		p.values = make([]float32, 4*n)
		for i := range n {
			copy(p.values[4*i:4*i+per], at(i))
		}
		return p, true
	}
	switch v := v.(type) {
	case float32:
		return one("f32", 4, v)
	case float64:
		return one("f32", 4, float32(v))
	case int:
		return one("f32", 4, float32(v))
	case bool:
		if v {
			return one("f32", 4, 1)
		}
		return one("f32", 4, 0)
	case geom.Vec2:
		return one("vec2f", 8, v.X, v.Y)
	case geom.Vec3:
		return one("vec3f", 12, v.X, v.Y, v.Z)
	case geom.Vec4:
		return one("vec4f", 16, v.X, v.Y, v.Z, v.W)
	case color.Color:
		return one("vec4f", 16, v.R, v.G, v.B, v.A)
	case []float32:
		return many("f32", ".x", len(v), 1, func(i int) []float32 { return []float32{v[i]} })
	case []geom.Vec2:
		return many("vec2f", ".xy", len(v), 2, func(i int) []float32 { return []float32{v[i].X, v[i].Y} })
	case []geom.Vec3:
		return many("vec3f", ".xyz", len(v), 3, func(i int) []float32 { return []float32{v[i].X, v[i].Y, v[i].Z} })
	case []geom.Vec4:
		return many("vec4f", "", len(v), 4, func(i int) []float32 { return []float32{v[i].X, v[i].Y, v[i].Z, v[i].W} })
	}
	return param{}, false
}

// pack writes ps into b, a Params struct's bytes.
func pack(b []byte, ps []param) {
	clear(b)
	for _, p := range ps {
		for i, f := range p.values {
			binary.LittleEndian.PutUint32(b[p.offset+4*i:], math.Float32bits(f))
		}
	}
}

// effectInputs is the EmberInputs struct of effectModule.
type effectInputs struct {
	Size, OutputSize [2]float32
	DepthRange       [3]float32
	Time             float32
}

// effectModule is the WGSL module running an effect whose source defines
// effect(vUV): a fullscreen triangle, the engine's inputs and the
// effect's own uniforms, copied into private variables under their own
// names before effect runs, and color and linearDepth. vUV runs 0..1 from
// the top-left, on a 2D layer and in a camera's post chain alike.
func effectModule(source string, ps []param) string {
	var b strings.Builder
	b.WriteString(`struct EmberInputs { uSize: vec2f, uOutputSize: vec2f, uDepthRange: vec3f, uTime: f32 };
struct EmberParams {
`)
	if len(ps) == 0 {
		b.WriteString("\tunused: vec4f,\n")
	}
	for _, p := range ps {
		fmt.Fprintf(&b, "\t%s: %s,\n", p.name, p.stored)
	}
	b.WriteString(`};
@group(0) @binding(0) var<uniform> emberInputs: EmberInputs;
@group(0) @binding(1) var uColor: texture_2d<f32>;
@group(0) @binding(2) var uColorSampler: sampler;
@group(0) @binding(3) var uDepth: texture_2d<f32>;
@group(0) @binding(4) var uDepthSampler: sampler;
@group(0) @binding(5) var<uniform> emberParams: EmberParams;
var<private> uSize: vec2f;
var<private> uOutputSize: vec2f;
var<private> uTime: f32;
var<private> uDepthRange: vec3f;
`)
	for _, p := range ps {
		fmt.Fprintf(&b, "var<private> %s: %s;\n", p.name, p.seen)
	}
	b.WriteString(`
fn color(uv: vec2f) -> vec4f { return textureSampleLevel(uColor, uColorSampler, uv, 0.0); }

fn linearDepth(uv: vec2f) -> f32 {
	let d = textureSampleLevel(uDepth, uDepthSampler, uv, 0.0).r;
	let n = uDepthRange.x;
	let f = uDepthRange.y;
	if (uDepthRange.z > 0.5) { return n + d * (f - n); }
	return n * f / (f - d * (f - n));
}

struct EmberVertex { @builtin(position) pos: vec4f, @location(0) uv: vec2f };

@vertex
fn ember_vs(@builtin(vertex_index) i: u32) -> EmberVertex {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return EmberVertex(vec4f(p.x * 2.0 - 1.0, 1.0 - p.y * 2.0, 0.0, 1.0), p);
}

@fragment
fn ember_fs(in: EmberVertex) -> @location(0) vec4f {
	uSize = emberInputs.uSize;
	uOutputSize = emberInputs.uOutputSize;
	uTime = emberInputs.uTime;
	uDepthRange = emberInputs.uDepthRange;
`)
	for _, p := range ps {
		if p.count == 0 || p.swizzle == "" {
			fmt.Fprintf(&b, "\t%s = emberParams.%s;\n", p.name, p.name)
			continue
		}
		fmt.Fprintf(
			&b,
			"\tfor (var i = 0u; i < %du; i++) { %s[i] = emberParams.%s[i]%s; }\n",
			p.count,
			p.name,
			p.name,
			p.swizzle,
		)
	}
	b.WriteString("\treturn effect(in.uv);\n}\n\n")
	b.WriteString(source)
	return b.String()
}

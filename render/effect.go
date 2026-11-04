package render

import (
	"iter"
	"slices"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
)

// Effect is a fullscreen WGSL fragment effect: its source defines
//
//	fn effect(vUV: vec2f) -> vec4f
//
// where vUV runs 0..1 from the top-left of the output, as in 2D, on a layer
// and in a camera's post chain alike. Beside it the presenter declares:
//
//	uSize: vec2f                 // size of uColor in pixels
//	uOutputSize: vec2f           // size of the output in pixels
//	uTime: f32
//	uDepthRange: vec3f           // near, far, and 1 for an orthographic view
//	fn color(uv: vec2f) -> vec4f // the previous image (tonemapped, display space)
//	fn linearDepth(uv: vec2f) -> f32 // view distance in world units
//
// and each of the effect's own uniforms by the name it was set under: f32,
// vec2f, vec3f or vec4f, or an array of those for the slice setters. It
// sets them each frame, so animate an effect by setting them: SetFloat,
// SetVec2, SetVec3, SetVec4 and SetColor, and the slice setters.
type Effect struct {
	name     string
	shader   string
	screen   bool
	disabled bool
	uniforms map[string]any
	err      error
}

// EffectOption sets up an Effect.
type EffectOption func(*Effect)

// NewEffect returns an enabled effect running shader, a WGSL source
// defining fn effect(vUV: vec2f) -> vec4f, where vUV runs 0–1 from the
// top-left, as in 2D.
func NewEffect(shader string, opts ...EffectOption) *Effect {
	e := &Effect{name: "effect", shader: shader, uniforms: map[string]any{}}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Name names an effect in its error messages.
func Name(name string) EffectOption { return func(e *Effect) { e.name = name } }

// Screen makes an effect render at window resolution and end a post
// chain: it is the place to upscale in a custom way. Effects after it are
// skipped. A 2D layer ignores it.
func Screen() EffectOption { return func(e *Effect) { e.screen = true } }

func (e *Effect) Name() string            { return e.name }
func (e *Effect) Shader() string          { return e.shader }
func (e *Effect) SetShader(shader string) { e.shader = shader }
func (e *Effect) Screen() bool            { return e.screen }
func (e *Effect) Enabled() bool           { return !e.disabled }
func (e *Effect) SetEnabled(on bool)      { e.disabled = !on }

func (e *Effect) SetFloat(name string, v float32)     { e.uniforms[name] = v }
func (e *Effect) SetVec2(name string, v geom.Vec2)    { e.uniforms[name] = v }
func (e *Effect) SetVec3(name string, v geom.Vec3)    { e.uniforms[name] = v }
func (e *Effect) SetVec4(name string, v geom.Vec4)    { e.uniforms[name] = v }
func (e *Effect) SetColor(name string, c color.Color) { e.uniforms[name] = c }
func (e *Effect) SetFloats(name string, v []float32)  { e.uniforms[name] = slices.Clone(v) }
func (e *Effect) SetVec2s(name string, v []geom.Vec2) { e.uniforms[name] = slices.Clone(v) }
func (e *Effect) SetVec3s(name string, v []geom.Vec3) { e.uniforms[name] = slices.Clone(v) }
func (e *Effect) SetVec4s(name string, v []geom.Vec4) { e.uniforms[name] = slices.Clone(v) }

// Uniform is the value last set for name.
func (e *Effect) Uniform(name string) (any, bool) {
	v, ok := e.uniforms[name]
	return v, ok
}

// Uniforms are every uniform set, by name; presenters read them.
func (e *Effect) Uniforms() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for name, v := range e.uniforms {
			if !yield(name, v) {
				return
			}
		}
	}
}

// Err is why the shader failed to compile, nil once it works. A
// failed effect is skipped until its shader changes.
func (e *Effect) Err() error { return e.err }

// SetErr records the outcome of compiling the shader; presenters call it.
func (e *Effect) SetErr(err error) { e.err = err }

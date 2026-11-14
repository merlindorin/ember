package render_test

import (
	"errors"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

func TestPostChainStopsAtTheFirstScreenEffect(t *testing.T) {
	var p render.Post
	if !p.Plain() || len(p.Active()) != 0 {
		t.Fatal("an empty post chain is plain")
	}
	grade := p.Add(render.NewEffect("", render.Name("grade")))
	off := p.Add(render.NewEffect("", render.Name("off")))
	off.SetEnabled(false)
	present := p.Add(render.NewEffect("", render.Name("present"), render.Screen()))
	p.Add(render.NewEffect("", render.Name("after")))
	got := p.Active()
	if len(got) != 2 || got[0] != grade || got[1] != present {
		t.Fatalf("active %v", got)
	}
	off.SetEnabled(true)
	present.SetEnabled(false)
	if again := p.Active(); len(again) != 3 || again[1] != off || again[2].Name() != "after" {
		t.Fatalf("active %v", again)
	}
	grade.SetEnabled(false)
	off.SetEnabled(false)
	p.Effects = p.Effects[:3]
	if !p.Plain() {
		t.Fatal("no active effect and linear upscale is plain")
	}
	p.Upscale = render.FilterNearest
	if p.Plain() {
		t.Fatal("nearest upscaling needs the chain")
	}
}

func TestSceneSize(t *testing.T) {
	var p *render.Post
	if w, h := p.SceneSize(geom.V2(1280, 720), 0.5); w != 640 || h != 360 {
		t.Fatalf("default scales the framebuffer: %dx%d", w, h)
	}
	q := &render.Post{Resolution: func(fb geom.Vec2) (int, int) { return int(fb.X) / 8, 0 }}
	if w, h := q.SceneSize(geom.V2(1280, 720), 1); w != 160 || h != 1 {
		t.Fatalf("custom resolution, clamped to 1: %dx%d", w, h)
	}
}

func TestAnEffectKeepsWhyItsShaderFailed(t *testing.T) {
	e := render.NewEffect("fn effect(vUV: vec2f) -> vec4f { return vec4f(1) }", render.Name("typo"))
	if e.Err() != nil {
		t.Fatal("no error before it is compiled")
	}
	e.SetErr(errors.New("1:52: expected ';'"))
	if e.Err() == nil || e.Err().Error() != "1:52: expected ';'" {
		t.Fatalf("the presenter's error: %v", e.Err())
	}
	e.SetErr(nil)
	if e.Err() != nil {
		t.Fatal("cleared once a new shader compiles")
	}
}

func TestAnEffectKeepsTheUniformsSetLast(t *testing.T) {
	e := render.NewEffect("fn effect(vUV: vec2f) -> vec4f { return color(vUV); }")
	if e.Name() != "effect" || !e.Enabled() || e.Screen() {
		t.Fatal("defaults: named effect, enabled, in the scene's resolution")
	}
	e.SetFloat("uLevels", 4)
	e.SetVec4s("uRipples", []geom.Vec4{{X: 1}, {X: 2}})
	e.SetFloat("uLevels", 6)
	if v, ok := e.Uniform("uLevels"); !ok || v != float32(6) {
		t.Fatalf("uLevels %v", v)
	}
	n := 0
	for range e.Uniforms() {
		n++
	}
	if n != 2 {
		t.Fatalf("%d uniforms", n)
	}
}

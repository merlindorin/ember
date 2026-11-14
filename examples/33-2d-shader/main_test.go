package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
)

func click(w interface{ RunFrames(int) }, win *platform.Window, at geom.Vec2) {
	win.Input().SetMouse(at)
	win.Input().SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	win.Input().SetMouseButton(input.MouseLeft, false)
	w.RunFrames(1)
}

func TestAClickStartsARippleTheShaderReads(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 384))
	effect := render.NewEffect(rippleShader)
	meadow := twod.NewLayer(0, twod.WithEffect(effect))
	win.AppendChild(meadow)
	w := ember.New().AddEntities(win).AddSystems(&Ripples{effect: effect, window: win})

	click(w, win, geom.V2(320, 96))
	r := rippleOf(t, effect, 0)
	if r.X != 0.5 || r.Y != 0.25 || r.W != 1 || r.Z <= 0 {
		t.Fatalf("the ripple starts where clicked, as a fraction of the window, and ages: %v", r)
	}
	if off := rippleOf(t, effect, 1); off.W != 0 {
		t.Fatalf("the other ripples are off: %v", off)
	}

	w.RunFrames(int(lifetime*60) + 1)
	if ended := rippleOf(t, effect, 0); ended.W != 0 {
		t.Fatalf("a ripple ends after its lifetime: %v", ended)
	}
}

func TestAFifthClickReplacesTheOldest(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 384))
	effect := render.NewEffect(rippleShader)
	win.Add(twod.NewLayer(0, twod.WithEffect(effect)))
	w := ember.New().AddEntities(win).AddSystems(&Ripples{effect: effect, window: win})

	for i := range 5 {
		click(w, win, geom.V2(float32(64*(i+1)), 192))
	}
	if r := rippleOf(t, effect, 0); r.X != 320.0/640 || r.Z > 0.05 {
		t.Fatalf("the fifth ripple took the first one's place, just started: %v", r)
	}
}

// rippleOf is the i-th ripple the shader was last given.
func rippleOf(t *testing.T, e *render.Effect, i int) geom.Vec4 {
	t.Helper()
	v, _ := e.Uniform("uRipples")
	ripples, ok := v.([]geom.Vec4)
	if !ok {
		t.Fatalf("uRipples is a %T", v)
	}
	return ripples[i]
}

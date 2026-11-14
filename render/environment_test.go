package render_test

import (
	"encoding/json"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

func TestPostEffectsAndFogAreOptIn(t *testing.T) {
	e := render.NewEnvironment()
	if s, _ := e.Bloom(); s != 0 || e.Vignette() != 0 {
		t.Fatal("bloom and vignette are off by default")
	}
	if _, d := e.Fog(); d != 0 {
		t.Fatal("fog is off by default")
	}
	if _, _, i := e.Sun(); i <= 0 || e.Exposure() != 1 {
		t.Fatal("a neutral daylight sun and exposure")
	}
	if e.Shadows() <= 0 {
		t.Fatal("shadows are on")
	}
}

func TestEnvironmentOptionsAndSetters(t *testing.T) {
	down := geom.V3(0, -1, 0)
	e := render.NewEnvironment(
		render.Sun(down, color.Red, 3), render.Shadows(12), render.Ambient(color.Blue, color.Black, 0.5),
		render.Fog(color.Gray(0.5), 0.02), render.Exposure(1.5), render.Bloom(0.6, 1.2), render.Vignette(0.25),
		render.ColorGrade(1.1, 0.9, color.White, color.Red),
	)
	if d, c, i := e.Sun(); d != down || c != color.Red || i != 3 {
		t.Fatalf("sun %v %v %v", d, c, i)
	}
	if e.Shadows() != 12 {
		t.Fatal("shadows")
	}
	if sky, ground, a := e.Ambient(); sky != color.Blue || ground != color.Black || a != 0.5 {
		t.Fatal("ambient")
	}
	if c, d := e.Fog(); c != color.Gray(0.5) || d != 0.02 {
		t.Fatal("fog")
	}
	if s, th := e.Bloom(); s != 0.6 || th != 1.2 || e.Vignette() != 0.25 || e.Exposure() != 1.5 {
		t.Fatal("post")
	}
	if s, c, lo, hi := e.ColorGrade(); s != 1.1 || c != 0.9 || lo != color.White || hi != color.Red {
		t.Fatal("grade")
	}
	e.SetFog(color.Black, 0)
	e.SetSun(geom.V3(1, -1, 0).Normalize(), color.White, 1)
	e.SetShadows(0)
	if _, d := e.Fog(); d != 0 {
		t.Fatal("SetFog")
	}
	if e.Shadows() != 0 {
		t.Fatal("SetShadows(0) turns them off")
	}
	if _, _, i := e.Sun(); i != 1 {
		t.Fatal("SetSun")
	}
}

func TestEnvironmentJSONAndLook(t *testing.T) {
	a := render.NewEnvironment(
		render.Fog(color.Red, 0.03),
		render.Bloom(0.4, 1),
		render.Sun(geom.V3(0, -1, 0), color.White, 2),
	)
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	_ = json.Unmarshal(data, &keys)
	if _, ok := keys["fog_density"]; !ok {
		t.Fatalf("snake_case keys: %s", data)
	}
	var b render.Environment
	if err = json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	if c, d := b.Fog(); c != color.Red || d != 0.03 {
		t.Fatal("round trip")
	}
	night := render.NewEnvironment(render.Sun(geom.V3(1, 0, 0), color.Blue, 0.1))
	night.CopyLook(a)
	if _, d := night.Fog(); d != 0.03 {
		t.Fatal("the look is copied")
	}
	if _, c, i := night.Sun(); c != color.Blue || i != 0.1 {
		t.Fatal("but not the sun")
	}
}

func TestCopyingALookKeepsTheEntity(t *testing.T) {
	e, night := render.NewEnvironment(), render.NewEnvironment(render.Fog(color.Black, 0.2))
	id := e.ID()
	e.CopyLook(night)
	if e.ID() != id || e.ID() == night.ID() {
		t.Fatal("the environment stays the same entity")
	}
	if _, d := e.Fog(); d != 0.2 {
		t.Fatal("and takes the look")
	}
}

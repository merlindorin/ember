package main

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
)

func TestCornersAreColouredByPosition(t *testing.T) {
	want := map[geom.Vec3]color.Color{
		geom.V3(-0.5, -0.5, -0.5): color.RGB(0, 0, 0),
		geom.V3(0.5, 0.5, 0.5):    color.RGB(1, 1, 1),
		geom.V3(0.5, -0.5, -0.5):  color.RGB(1, 0, 0),
		geom.V3(-0.5, 0.5, -0.5):  color.RGB(0, 1, 0),
		geom.V3(-0.5, -0.5, 0.5):  color.RGB(0, 0, 1),
	}
	seen := map[geom.Vec3]bool{}
	for _, v := range rgbCube().Vertices {
		if c, ok := want[v.Pos]; ok {
			seen[v.Pos] = true
			if v.Color != c {
				t.Fatalf("corner %v is %v, want %v", v.Pos, v.Color, c)
			}
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("found %d of %d corners", len(seen), len(want))
	}
}

func TestTheCubeSpinsUnlitOnItsBackground(t *testing.T) {
	a := helloCube()
	var cube *Cube
	var win *platform.Window
	for _, e := range a.Entities() {
		switch e := e.(type) {
		case *Cube:
			cube = e
		case *platform.Window:
			win = e
		}
	}
	if cube == nil || !cube.Material.Unlit {
		t.Fatal("an unlit cube shows its vertex colours as they are")
	}
	if win == nil || win.Title() != "ember · hello cube" || win.Background() != color.Hex("#202028") {
		t.Fatalf("window %q on %v", win.Title(), win.Background())
	}
	start := cube.Rotation
	a.RunFrames(60)
	r := cube.Rotation
	if d := r.X*start.X + r.Y*start.Y + r.Z*start.Z + r.W*start.W; d > 0.999 {
		t.Fatalf("a second of spin turns it visibly: dot %v", d)
	}
}

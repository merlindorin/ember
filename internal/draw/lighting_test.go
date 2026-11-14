package draw_test

import (
	"testing"
	"unsafe"

	"github.com/lantervale/ember/internal/draw"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
)

func TestALitListMarksItsVerticesWithWhereTheyReadTheLight(t *testing.T) {
	size := geom.V2(200, 100)
	toLight := draw.ToLight(size)
	r := draw.LightRect(size)
	if p := toLight.Xform(geom.V2(r.X, r.Y)); p != (geom.Vec2{}) {
		t.Fatalf("the light rectangle's corner is the picture's 0, 0: %v", p)
	}
	if p := toLight.Xform(geom.V2(r.X+r.W, r.Y+r.H)); p.Sub(geom.V2(1, 1)).Len() > 1e-6 {
		t.Fatalf("and its far corner 1, 1: %v", p)
	}
	var l draw.List
	l.Reset(0)
	l.SetLight(draw.LitPerPixel, toLight, geom.Vec2{})
	l.Rect(geom.Rect{X: 10, Y: 20, W: 4, H: 4}, color.White)
	l.SetLight(draw.LitAtAnchor, toLight, geom.V2(100, 50))
	l.Rect(geom.Rect{X: 30, Y: 20, W: 4, H: 4}, color.White)
	l.SetLight(draw.Unlit, toLight, geom.Vec2{})
	l.Rect(geom.Rect{X: 50, Y: 20, W: 4, H: 4}, color.White)
	l.SetLight(draw.LitPerPixel, toLight, geom.Vec2{})
	l.Picture(geom.Rect{W: 200, H: 100}, draw.NewPass(), draw.Alpha, nil)
	vs := l.Vertices
	for _, v := range vs[:4] {
		if v.Lit() != 1 || v.LightUV != toLight.Xform(v.Pos) {
			t.Fatalf("per pixel, each vertex where it stands: %+v", v)
		}
	}
	for _, v := range vs[4:8] {
		if v.Lit() != 1 || v.LightUV != toLight.Xform(geom.V2(100, 50)) {
			t.Fatalf("at an anchor, every vertex the anchor's: %+v", v)
		}
	}
	for _, v := range vs[8:] {
		if v.Lit() != 0 {
			t.Fatalf("unlit, and a pass's picture is never lit again: %+v", v)
		}
	}
	l.Reset(0)
	l.Rect(geom.Rect{W: 1, H: 1}, color.White)
	if l.Vertices[0].Lit() != 0 {
		t.Fatal("a list starts each frame unlit")
	}
}

func TestALitAroundListMarksItsVerticesWithTheBoxsMiddleAndHalfSize(t *testing.T) {
	size := geom.V2(200, 100)
	toLight := draw.ToLight(size)
	var l draw.List
	l.Reset(0)
	l.SetLightAround(toLight, geom.Rect{X: 90, Y: 40, W: 20, H: 10})
	l.Rect(geom.Rect{X: 30, Y: 20, W: 4, H: 4}, color.White)
	mid := toLight.Xform(geom.V2(100, 45))
	half := toLight.Xform(geom.V2(110, 50)).Sub(mid)
	for _, v := range l.Vertices {
		if v.Lit() != 2 || v.LightUV.Sub(mid).Len() > 1e-6 || v.LightSpread.Sub(half).Len() > 1e-6 {
			t.Fatalf("every vertex the box's middle %v and half-size %v: %+v", mid, half, v)
		}
	}
}

func TestALitFacingDrawingIsLitRoundItsBoxAndThenAsBefore(t *testing.T) {
	toLight := draw.ToLight(geom.V2(200, 100))
	var l draw.List
	l.Reset(0)
	l.SetLight(draw.Unlit, toLight, geom.Vec2{})
	l.SetLightFacing(geom.Rect{X: 90, Y: 40, W: 20, H: 10})
	l.Rect(geom.Rect{W: 4, H: 4}, color.White)
	if v := l.Vertices[0]; v.Lit() != 0 {
		t.Fatalf("on a list that isn't lit per pixel, nothing changes: %+v", v)
	}
	l.SetLight(draw.LitPerPixel, toLight, geom.Vec2{})
	before := l.Light()
	l.SetLightFacing(geom.Rect{X: 90, Y: 40, W: 20, H: 10})
	l.Rect(geom.Rect{W: 4, H: 4}, color.White)
	l.RestoreLight(before)
	l.Rect(geom.Rect{X: 10, W: 4, H: 4}, color.White)
	for _, v := range l.Vertices[4:8] {
		if v.Lit() != 3 || v.LightUV.Sub(toLight.Xform(geom.V2(100, 45))).Len() > 1e-6 {
			t.Fatalf("lit facing, round the box's middle: %+v", v)
		}
	}
	for _, v := range l.Vertices[8:] {
		if v.Lit() != 1 || v.LightUV != toLight.Xform(v.Pos) {
			t.Fatalf("restored, per pixel again: %+v", v)
		}
	}
}

func TestAVertexIsSixtyEightBytesItsLightModeInItsSpread(t *testing.T) {
	if n := unsafe.Sizeof(draw.Vertex{}); n != 68 {
		t.Fatalf("every 2D vertex the GPU reads is %d bytes, not 68", n)
	}
	if (draw.Vertex{}).Lit() != 0 {
		t.Fatal("a vertex nothing lit is drawn as it is")
	}
}

func TestAListShadedBySomeGroupsMarksItsVerticesAndTheShadesItNeeds(t *testing.T) {
	toLight := draw.ToLight(geom.V2(200, 100))
	var l draw.List
	l.Reset(0)
	l.SetLight(draw.LitPerPixel, toLight, geom.Vec2{})
	l.Rect(geom.Rect{W: 4, H: 4}, color.White)
	l.SetShadedBy(0b011)
	l.Rect(geom.Rect{X: 10, W: 4, H: 4}, color.White)
	l.SetLightAround(toLight, geom.Rect{X: 90, Y: 40, W: 20, H: 10})
	l.Rect(geom.Rect{X: 20, W: 4, H: 4}, color.White)
	l.SetShadedBy(0)
	l.SetLight(draw.LitAtAnchor, toLight, geom.V2(100, 50))
	l.Rect(geom.Rect{X: 30, W: 4, H: 4}, color.White)
	want := []struct {
		lit   int
		shade uint8
	}{{1, draw.AllGroups}, {1, 0b011}, {2, 0b011}, {1, 0}}
	for i, w := range want {
		v := l.Vertices[4*i]
		if v.Lit() != w.lit || v.Shade() != w.shade {
			t.Fatalf("drawing %d: lit %d shaded by %03b, want %d and %03b", i, v.Lit(), v.Shade(), w.lit, w.shade)
		}
	}
	if spread := l.Vertices[8].LightSpread.X - 2*float32(draw.AllGroups&^0b011); spread <= 0 || spread > 1 {
		t.Fatalf("the spread is kept under the groups: %v", l.Vertices[8].LightSpread)
	}
	if l.Shades != draw.KeysOf(0, 0b100, 0b111) {
		t.Fatalf("the shades drawn are marked: %08b", l.Shades)
	}
	l.SetLight(draw.Unlit, toLight, geom.Vec2{})
	l.SetShadedBy(0b001)
	l.Rect(geom.Rect{W: 4, H: 4}, color.White)
	if l.Shades.Has(0b110) {
		t.Fatal("an unlit drawing needs no shade")
	}
	l.Reset(0)
	if l.Shades != (draw.Keys{}) {
		t.Fatal("Reset forgets them")
	}
}

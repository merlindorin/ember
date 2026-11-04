package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheLinesAndShapesDraw(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	miter := twod.NewLine(zigzag(), twod.Width(14), twod.Joints(twod.Miter))
	round := twod.NewLine(
		zigzag(),
		twod.Width(14),
		twod.Joints(twod.RoundJoint),
		twod.Caps(twod.RoundCap, twod.RoundCap),
	)
	filled := twod.NewPolygon(arrow(), twod.At(370, 200))
	w := ember.New().AddEntities(win)
	count := func(e twod.Entity) int {
		win.AppendChild(e)
		w.AddEntities(e)
		w.RunFrames(1)
		n := len(draw.FrontFor(w, win).Unindexed())
		w.RemoveEntity(e)
		return n
	}
	sharp, smooth := count(miter), count(round)
	if sharp == 0 || smooth <= sharp {
		t.Fatalf("round joints and caps take more triangles: %d, %d", sharp, smooth)
	}
	if n := count(filled); n != (len(arrow())-2)*3 {
		t.Fatalf("a concave polygon of %d points fills with %d triangles: %d vertices", len(arrow()), len(arrow())-2, n)
	}
}

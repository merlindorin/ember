package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheCornersKeepTheirSizeAndTheMiddleStretches(t *testing.T) {
	panel, err := twod.LoadTexture(files, "panel.png", twod.Pixelated())
	if err != nil {
		t.Fatal(err)
	}
	if panel.Size() != geom.V2(24, 24) {
		t.Fatalf("panel.png is %v", panel.Size())
	}
	win := platform.NewWindow(platform.Size(640, 360))
	win.AppendChild(
		twod.NewNinePatch(
			panel,
			twod.Margins(8, 8, 8, 8),
			twod.Size(64, 48),
			twod.At(40, 60),
			twod.Pivot(0, 0),
			twod.Scale(3, 3),
		),
	)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)

	vs := draw.FrontFor(w, win).Unindexed()
	if len(vs) != 9*6 {
		t.Fatalf("a nine-patch is nine quads: %d vertices", len(vs))
	}
	lo, hi := vs[0].Pos, vs[0].Pos
	for _, v := range vs {
		lo, hi = geom.V2(min(lo.X, v.Pos.X), min(lo.Y, v.Pos.Y)), geom.V2(max(hi.X, v.Pos.X), max(hi.Y, v.Pos.Y))
	}
	if size := hi.Sub(lo); size != geom.V2(192, 144) {
		t.Fatalf("64×48 points scaled 3 times: %v", size)
	}
	corner := vs[0:6]
	cw := max(
		corner[0].Pos.X,
		corner[1].Pos.X,
		corner[2].Pos.X,
	) - min(
		corner[0].Pos.X,
		corner[1].Pos.X,
		corner[2].Pos.X,
	)
	if cw != 24 {
		t.Fatalf("a corner stays 8 texture pixels, 24 points at Scale(3): %v", cw)
	}
}

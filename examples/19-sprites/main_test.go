package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestTheCratesDrawInTheWindow(t *testing.T) {
	crate, err := twod.LoadTexture(files, "crate.png", twod.Pixelated())
	if err != nil {
		t.Fatal(err)
	}
	if crate.Size() != geom.V2(16, 16) {
		t.Fatalf("crate.png is %v", crate.Size())
	}
	win := platform.NewWindow(platform.Size(640, 360))
	plain := twod.NewSprite(crate, twod.At(160, 180), twod.Scale(6, 6))
	mossy := twod.NewSprite(
		crate,
		twod.At(480, 180),
		twod.Scale(6, 6),
		twod.Flip(true, false),
		twod.Tint(color.Hex("#8fd18a")),
	)
	win.AppendChild(plain)
	win.AppendChild(mossy)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)

	vs := draw.FrontFor(w, win).Unindexed()
	if len(vs) != 12 {
		t.Fatalf("two sprites, two quads: %d vertices", len(vs))
	}
	if vs[0].Pos != geom.V2(112, 132) {
		t.Fatalf("the first crate is 96 points wide, centred on 160, 180: %v", vs[0].Pos)
	}
	if vs[6].Color != color.Hex("#8fd18a") || vs[6].UV.X != 1 {
		t.Fatalf("the second is tinted and mirrored: %+v", vs[6])
	}
}

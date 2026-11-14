package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestTheWalkerGoesRoundTheLoopAndBack(t *testing.T) {
	walker, err := twod.LoadTexture(files, "walker.png")
	if err != nil {
		t.Fatal(err)
	}
	path := newPath(walker)
	var runner *twod.Sprite
	for _, n := range path.GetSceneNode().ChildNodes() {
		if s, ok := n.Entity().(*twod.Sprite); ok {
			runner = s
		}
	}
	w := ember.New().AddEntities(platform.NewWindow().Add(path))
	for range 180 {
		w.Tick(1.0 / 60)
	}
	if at := runner.Position(); at.Dist(geom.ClosedCatmullRom(loop(), 0.5)) > 2 {
		t.Fatalf("half way round after three seconds: %v", at)
	}
	for range 180 {
		w.Tick(1.0 / 60)
	}
	if at := runner.Position(); at.Dist(loop()[0]) > 2 {
		t.Fatalf("back at the start after six: %v", at)
	}
}

func TestEveryFamilyHasARow(t *testing.T) {
	if len(easings()) != 8 {
		t.Fatalf("Linear and the seven families: %d", len(easings()))
	}
	for _, e := range easings() {
		if e.f(0) > 1e-3 || e.f(1) < 0.999 {
			t.Fatalf("%s runs from 0 to 1", e.name)
		}
	}
}

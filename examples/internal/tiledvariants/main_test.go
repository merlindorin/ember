package main

import (
	"os"
	"testing"

	"github.com/merlindorin/ember"
)

func TestTheFixtureLoadsWithItsPalettesAndMasks(t *testing.T) {
	win, err := scene(os.DirFS("../../../twod/tiled/variants"))
	if err != nil {
		t.Fatal(err)
	}
	ember.New().AddEntities(win).RunFrames(2)
}

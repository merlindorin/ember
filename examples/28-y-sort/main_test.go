package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestTheHeroSortsWithTheTreesAndStopsAtTheirTrunks(t *testing.T) {
	ground, things, err := loadTilesets()
	if err != nil {
		t.Fatal(err)
	}
	grassland := twod.NewTileMap(ground, twod.ASCII(field, map[rune]int{'.': grass}), twod.CellSize(32, 32))
	garden := twod.NewLayer(0, twod.SortByY())
	hero, err := newHero(grassland.Center(8, 9))
	if err != nil {
		t.Fatal(err)
	}
	garden.Add(hero)
	trees := twod.NewTileMap(things, twod.ASCII(objects, objectsLegend()), twod.CellSize(32, 32), twod.TallIn(garden))
	if n := len(trees.TallSprites()); n != 17 {
		t.Fatalf("12 trees and 5 pillars stand in the garden: %d", n)
	}
	for _, s := range trees.TallSprites() {
		if s.GetSceneNode().ParentNode() != garden.GetSceneNode() {
			t.Fatal("beside the hero, to sort with it")
		}
	}

	space := collide.NewSpace()
	space.AddSolid(tiles.Of(trees))
	win := platform.NewWindow(platform.Size(640, 384))
	win.AppendChild(grassland)
	win.AppendChild(garden)
	win.AppendChild(trees)
	keys := win.Input()
	w := ember.New().AddEntities(win).AddSystems(collide.NewTracker(space), &Walk{hero: hero, keys: keys, space: space})
	keys.SetKey(input.KeyUp, true)
	for range 120 {
		w.Tick(1.0 / 60)
	}
	if y := hero.Position().Y; y < 8*32+8-0.1 || y > 8*32+8+0.1 {
		t.Fatalf(
			"walking up, the top of the hero's 8-deep body stops at the bottom of the trunk's cell (8, 7): feet at %v",
			y,
		)
	}
}

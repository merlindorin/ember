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

func TestTheTiledLevelSpawnsItsThingsAndItsWallsBlock(t *testing.T) {
	level, err := twod.LoadTiled(files, "level.tmj", tiles.Properties())
	if err != nil {
		t.Fatal(err)
	}
	walls, ok := level.TileLayer("walls")
	if !ok || len(level.TileLayers()) != 2 {
		t.Fatal("two tile layers, ground and walls")
	}
	if x, y, _ := walls.Cell(walls.Center(0, 0)); walls.At(x, y) != 2 || !tiles.Of(walls).Solid(x, y) {
		t.Fatal("the wall tile is solid by its property in tiles.tsj")
	}
	things := twod.NewLayer(1, twod.SortByY())
	hero, err := spawn(things, level, level.Objects("things"))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(things.GetSceneNode().ChildNodes()); n != 4 {
		t.Fatalf("the hero and three pillars: %d", n)
	}
	space := collide.NewSpace()
	win := platform.NewWindow(platform.Size(640, 384))
	for _, l := range level.TileLayers() {
		win.AppendChild(l)
		space.AddSolid(tiles.Of(l))
	}
	win.AppendChild(things)
	keys := win.Input()
	w := ember.New().AddEntities(win).AddSystems(collide.NewTracker(space), &Walk{hero: hero, keys: keys, space: space})
	keys.SetKey(input.KeyLeft, true)
	for range 180 {
		w.Tick(1.0 / 60)
	}
	if x := hero.Position().X; x < 39.9 || x > 40.1 {
		t.Fatalf("the left wall stops the hero: x %v", x)
	}
}

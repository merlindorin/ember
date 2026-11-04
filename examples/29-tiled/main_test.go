package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheTiledLevelSpawnsItsThingsAndItsWallsBlock(t *testing.T) {
	level, err := twod.LoadTiled(files, "level.tmj")
	if err != nil {
		t.Fatal(err)
	}
	walls, ok := level.Layer("walls")
	if !ok || len(level.Layers()) != 2 {
		t.Fatal("two tile layers, ground and walls")
	}
	if x, y, _ := walls.Cell(walls.Center(0, 0)); walls.At(x, y) != 2 || !level.Tilesets()[0].Solid(2) {
		t.Fatal("the wall tile is solid by its property in tiles.tsj")
	}
	things := twod.NewLayer(1, twod.SortByY())
	hero, err := spawn(things, level.Objects("things"))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(things.GetSceneNode().ChildNodes()); n != 4 {
		t.Fatalf("the hero and three pillars: %d", n)
	}
	space := collide.NewSpace()
	win := platform.NewWindow(platform.Size(640, 384))
	for _, l := range level.Layers() {
		win.AppendChild(l)
		space.AddSolid(l)
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

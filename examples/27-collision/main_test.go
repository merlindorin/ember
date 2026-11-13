package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestWallsStopTheHeroUntilThePlateOpensTheDoor(t *testing.T) {
	sheet, err := twod.LoadTexture(files, "tiles.png")
	if err != nil {
		t.Fatal(err)
	}
	walls := twod.NewTileMap(
		twod.NewTileset(sheet, twod.Cell(16, 16), tiles.Solid(wall)),
		twod.ASCII(room, legend()),
		twod.CellSize(32, 32),
	)
	hero, err := newHero(walls.Center(3, 8))
	if err != nil {
		t.Fatal(err)
	}
	plate := twod.NewSprite(nil, twod.At(walls.Center(plateX, plateY).X, walls.Center(plateX, plateY).Y))
	plate.AppendChild(collide.NewBody(collide.Box(32, 32), collide.Trigger()))
	space := collide.NewSpace()
	space.AddSolid(tiles.Of(walls))
	win := platform.NewWindow(platform.Size(640, 384))
	win.AppendChild(walls)
	win.AppendChild(plate)
	win.AppendChild(hero)
	keys := win.Input()
	w := ember.New().
		AddEntities(win).
		AddSystems(collide.NewTracker(space), &Walk{hero: hero, keys: keys, space: space}, &Door{walls: walls})
	walk := func(k input.Key, seconds float32) {
		keys.SetKey(k, true)
		for range int(seconds * 60) {
			w.Tick(1.0 / 60)
		}
		keys.SetKey(k, false)
	}

	walk(input.KeyLeft, 2)
	if x := hero.Position().X; x < 71.9 || x > 72.1 {
		t.Fatalf("the left wall stops the hero's 16-wide body at its face: x %v", x)
	}
	if walls.At(doorX, doorY) != wall {
		t.Fatal("the door is shut")
	}
	walk(input.KeyRight, 4)
	if walls.At(doorX, doorY) != floor {
		t.Fatal("stepping on the plate opens the door")
	}
	if x := hero.Position().X; x < 9*32 {
		t.Fatalf("the hero walks through the open door into the other half: x %v", x)
	}
}

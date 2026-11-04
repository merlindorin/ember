package main

import (
	"encoding/json"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheHeroWalksAndTreesGrowUnderThePointer(t *testing.T) {
	tiles, err := twod.LoadTexture(files, "tiles.png")
	if err != nil {
		t.Fatal(err)
	}
	hero, err := newHero()
	if err != nil {
		t.Fatal(err)
	}
	garden := newGarden(tiles)
	eye := twod.NewCamera(twod.Follow(hero))
	win := platform.NewWindow(platform.Size(640, 400))
	win.Add(twod.NewLayer(1).Add(garden.trees, hero), garden.ghost, eye)
	in := win.Input()
	controls := newControls()
	w := ember.New().AddEntities(win).AddSystems(
		&Walk{hero: hero, in: in, controls: controls},
		&Plant{garden: garden, eye: eye, in: in, controls: controls},
	)
	step := func() {
		w.Tick(1.0 / 60)
	}

	in.SetKey(input.KeyD, true)
	for range 60 {
		step()
	}
	in.SetKey(input.KeyD, false)
	if x := hero.Position().X; x < 479 || x > 481 {
		t.Fatalf("D walks the hero 160 points right in a second: x %v", x)
	}

	in.ConnectGamepad(0, "pad")
	in.SetGamepadStick(0, input.LeftStick, geom.V2(-1, 0))
	for range 30 {
		step()
	}
	in.SetGamepadStick(0, input.LeftStick, geom.Vec2{})
	step()
	if x := hero.Position().X; x < 398 || x > 402 {
		t.Fatalf("the pad's stick walks the hero back through the same actions: x %v", x)
	}

	in.SetMouse(geom.V2(420, 100))
	in.AddWheel(geom.V2(0, -5))
	step()
	step()
	if z := eye.Zoom(); z < 1.49 || z > 1.51 {
		t.Fatalf("five lines up zoom in by half: %v", z)
	}
	under := eye.ScreenToWorld(in.Mouse())
	if garden.ghost.Position() != under {
		t.Fatalf("the ghost tree is on the world under the pointer: %v, want %v", garden.ghost.Position(), under)
	}
	in.SetMouseButton(input.MouseLeft, true)
	step()
	step()
	if n := len(garden.trees.Children()); n != 1 {
		t.Fatalf("one click plants one tree: %d trees", n)
	}
}

func TestRebindingsSaveAndLoadByName(t *testing.T) {
	controls := newControls()
	controls.Bind(PlantTree, input.KeySpace)
	saved, err := json.Marshal(controls)
	if err != nil {
		t.Fatal(err)
	}
	loaded := newControls()
	if err = json.Unmarshal(saved, loaded); err != nil {
		t.Fatal(err)
	}
	if b := loaded.Bindings(PlantTree); len(b) != 1 || b[0] != input.KeySpace {
		t.Fatalf("Space alone plants after a reload: %v", b)
	}
	if a, ok := loaded.Bound(input.KeyW); !ok || a != Up {
		t.Fatal("W still walks up")
	}
}

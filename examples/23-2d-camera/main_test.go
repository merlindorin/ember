package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheCameraFollowsTheHeroToTheLevelsEnd(t *testing.T) {
	hero, err := newHero()
	if err != nil {
		t.Fatal(err)
	}
	win := platform.NewWindow(platform.Size(640, 360))
	eye := twod.NewCamera(twod.Follow(hero), twod.Bounds(level()))
	win.AppendChild(hero)
	win.AppendChild(eye)
	keys := win.Input()
	w := ember.New().AddEntities(win).AddSystems(&Walk{hero: hero, keys: keys})

	w.Tick(1.0 / 60)
	if eye.Center() != geom.V2(800, 180) {
		t.Fatalf("the camera starts on the hero, the level as tall as the view: %v", eye.Center())
	}
	keys.SetKey(input.KeyLeft, true)
	for range 600 {
		w.Tick(1.0 / 60)
	}
	if hero.Position().X > 0 || hero.Position().Y != groundLine || eye.Center().X != 320 {
		t.Fatalf(
			"the hero walks off the left end on the ground, the camera stops at it: hero %v, camera %v",
			hero.Position(),
			eye.Center(),
		)
	}
	if flipX, _ := hero.Flip(); !flipX {
		t.Fatal("facing left")
	}
}

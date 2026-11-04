package main

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/world"
)

func TestTheFogLiftsRoundTheHero(t *testing.T) {
	hero, err := newHero(geom.V2(10*32+16, 6*32+16))
	if err != nil {
		t.Fatal(err)
	}
	fog := newFog()
	(&Reveal{fog: fog, hero: hero}).Update(world.Frame{})
	if !fog.seen[[2]int{10, 6}] || !fog.seen[[2]int{10 + sight, 6}] {
		t.Fatal("the hero sees his own cell and three cells away")
	}
	if fog.seen[[2]int{10 + sight, 6 + sight}] || fog.seen[[2]int{14, 6}] {
		t.Fatal("but not past his sight, in a circle")
	}
	hero.SetPosition(geom.V2(2*32, 6*32))
	(&Reveal{fog: fog, hero: hero}).Update(world.Frame{})
	if !fog.seen[[2]int{10, 6}] || !fog.seen[[2]int{2, 6}] {
		t.Fatal("what he has seen stays seen")
	}
}

func TestTheClearingIsBrightestAtTheGardensHeart(t *testing.T) {
	if !(level(10, 6) > level(6, 6) && level(6, 6) > level(0, 0)) || level(0, 0) != 0 {
		t.Fatalf("the light fades from the heart: %v, %v, %v", level(10, 6), level(6, 6), level(0, 0))
	}
}

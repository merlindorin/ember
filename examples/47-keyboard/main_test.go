package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestTheHeroWalksRunsAndIsNamed(t *testing.T) {
	hero, err := newHero()
	if err != nil {
		t.Fatal(err)
	}
	tag := twod.NewText("Hero")
	hero.Add(tag)
	win := platform.NewWindow(platform.Size(640, 360))
	win.Add(hero)
	naming := &Naming{window: win, tag: tag, name: "Hero"}
	in := win.Input()
	w := ember.New().AddEntities(win).AddSystems(naming, &Walk{hero: hero, in: in, naming: naming})
	hold := func(k input.Key, frames int) {
		in.SetKey(k, true)
		w.RunFrames(frames)
		in.SetKey(k, false)
		w.RunFrames(1)
	}

	start := hero.Position()
	hold(input.KeyD, 60)
	if d := hero.Position().X - start.X; d < 115 || d > 125 {
		t.Fatalf("D walks 120 points a second: %v", d)
	}
	in.SetKey(input.KeyRightShift, true)
	start = hero.Position()
	hold(input.KeyLeft, 60)
	in.SetKey(input.KeyRightShift, false)
	if d := start.X - hero.Position().X; d < 235 || d > 245 {
		t.Fatalf("either Shift runs at twice the speed: %v", d)
	}

	hold(input.KeyEnter, 1)
	hold(input.KeyBackspace, 1)
	for _, r := range "o" {
		in.AddText(r)
	}
	w.RunFrames(1)
	at := hero.Position()
	hold(input.KeyW, 10)
	if hero.Position() != at {
		t.Fatal("typing a name doesn't walk the hero")
	}
	in.AddText('n')
	w.RunFrames(1)
	if tag.Text() != "Heron_" {
		t.Fatalf("the tag shows the name being typed: %q", tag.Text())
	}
	hold(input.KeyEnter, 1)
	if tag.Text() != "Heron" {
		t.Fatalf("the name is what was typed: %q", tag.Text())
	}
}

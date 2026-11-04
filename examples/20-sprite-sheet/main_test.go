package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheHeroWalksWhereTheArrowsPoint(t *testing.T) {
	sheet, err := twod.LoadTexture(files, "walker.png")
	if err != nil {
		t.Fatal(err)
	}
	frames := twod.Strip(sheet, twod.Cell(16, 16))
	clips := twod.NewAnimations().
		Add("down", 8, frames[0:4], twod.Loop()).
		Add("side", 8, frames[8:12], twod.Loop())
	win := platform.NewWindow()
	hero := twod.NewSprite(sheet, twod.Animate(clips), twod.At(320, 180))
	win.AppendChild(hero)
	keys := win.Input()
	w := ember.New().AddEntities(win).AddSystems(&Walk{hero: hero, keys: keys})
	walk := func(k input.Key, frames int) {
		keys.SetKey(k, true)
		for range frames {
			w.Tick(1.0 / 60)
		}
		keys.SetKey(k, false)
	}

	walk(input.KeyRight, 20)
	clip, playing := hero.Playing()
	if x := hero.Position().X; x < 358 || x > 362 || clip != "side" || !playing || hero.Frame() != 2 {
		t.Fatalf("a third of a second right: x %v, clip %q playing %v frame %d", x, clip, playing, hero.Frame())
	}
	if flipX, _ := hero.Flip(); flipX {
		t.Fatal("facing right")
	}
	walk(input.KeyLeft, 1)
	if flipX, _ := hero.Flip(); !flipX {
		t.Fatal("walking left mirrors the side clip")
	}
	w.Tick(1.0 / 60)
	if _, still := hero.Playing(); still || hero.Frame() != 0 {
		t.Fatal("with no key held, it stops on the first frame")
	}
	walk(input.KeyDown, 1)
	if down, _ := hero.Playing(); down != "down" {
		t.Fatalf("walking down plays %q", down)
	}
}

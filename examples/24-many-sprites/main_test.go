package main

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestBunniesBounceInsideTheWindowAndSpaceAddsMore(t *testing.T) {
	bunny, err := twod.LoadTexture(files, "bunny.png", twod.Pixelated())
	if err != nil {
		t.Fatal(err)
	}
	win := platform.NewWindow(platform.Size(width, height))
	field, counter := twod.NewLayer(0), twod.NewText("")
	win.AppendChild(field)
	win.AppendChild(counter)
	bunnies := newBunnies(bunny, win, field, counter)
	w := ember.New().AddEntities(win).AddSystems(bunnies)

	w.RunFrames(120)
	if len(bunnies.all) != 500 || !strings.HasPrefix(counter.Text(), "500 bunnies") {
		t.Fatalf("five hundred bunnies to start: %d, %q", len(bunnies.all), counter.Text())
	}
	for _, b := range bunnies.all {
		if at := b.sprite.Position(); at.X < 0 || at.X > width || at.Y > height {
			t.Fatalf("a bunny left the window: %v", at)
		}
	}

	win.Input().SetKey(input.KeySpace, true)
	w.RunFrames(5)
	if len(bunnies.all) != 1000 {
		t.Fatalf("holding Space adds a hundred a frame: %d", len(bunnies.all))
	}
}

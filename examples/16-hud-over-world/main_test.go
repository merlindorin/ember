package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

func scene(t *testing.T) (*world.World, *platform.Window, []*walker, *ui.Node, *ui.Document) {
	t.Helper()
	sheet, err := twod.LoadTexture(files, "walker.png")
	if err != nil {
		t.Fatal(err)
	}
	win := platform.NewWindow(platform.Size(640, 384))
	walkers := newWalkers(sheet)
	hud, score := newHUD(walkers)
	for _, w := range walkers {
		win.Add(w.sprite)
	}
	win.Add(hud)
	w := ember.New().AddEntities(win).
		AddSystems(&Wander{walkers: walkers}, &Hit{walkers: walkers, mouse: win.Input(), score: score})
	return w, win, walkers, score, hud
}

func TestTagsFollowTheirWalkers(t *testing.T) {
	w, _, walkers, _, hud := scene(t)
	for range 90 {
		w.Tick(1.0 / 60)
	}
	for _, wk := range walkers {
		p, r := wk.sprite.Position(), border(hud, wk.tag)
		if d := r.X + r.W/2 - p.X; d < -1 || d > 1 {
			t.Errorf("tag centred %.1f points off its walker", d)
		}
		if d := r.Y + r.H - (p.Y - 20); d < -1 || d > 1 {
			t.Errorf("tag bottom %.1f points off 20 above its walker", d)
		}
	}
	if p := walkers[0].sprite.Position(); p.X < 200 || p.X > 220 {
		t.Fatalf("Ada walked 90 points of her route in 1.5 s: %v", p)
	}
}

func TestAClickHurtsOnlyTheWalkerUnderIt(t *testing.T) {
	w, win, walkers, score, _ := scene(t)
	w.Tick(1.0 / 60)
	mouse := win.Input()
	mouse.SetMouse(walkers[1].sprite.Position())
	mouse.SetMouseButton(input.MouseLeft, true)
	w.Tick(1.0 / 60)
	mouse.SetMouseButton(input.MouseLeft, false)
	w.Tick(1.0 / 60)
	if walkers[1].health != 0.75 || walkers[0].health != 1 || walkers[2].health != 1 {
		t.Fatalf("health %v %v %v", walkers[0].health, walkers[1].health, walkers[2].health)
	}
	if score.Text() != "Hits 1" {
		t.Fatalf("score %q", score.Text())
	}
}

// border is n's box in the window: its content box, as tags have no
// padding or border.
func border(doc *ui.Document, n *ui.Node) geom.Rect { r, _ := doc.BoxOf(n); return r }

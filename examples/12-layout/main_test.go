package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
)

func TestTheMenuFollowsTheWindow(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 400))
	menu := newMenu()
	win.Add(menu)
	w := ember.New().AddEntities(win)
	title, levels, footer := menu.ByID("title"), menu.ByID("levels"), menu.ByID("footer")
	box := func(n *ui.Node) geom.Rect { r, _ := menu.BoxOf(n); return r }

	for _, size := range []geom.Vec2{geom.V2(640, 400), geom.V2(900, 560)} {
		win.SetSize(size)
		w.RunFrames(2)

		if c := box(title).Center().X; c < size.X/2-1 || c > size.X/2+1 {
			t.Errorf("%v: the title is centred: %v", size, box(title))
		}
		if c := box(levels).Center(); c.Dist(geom.V2(size.X/2, c.Y)) > 1 {
			t.Errorf("%v: the levels are centred across: %v", size, box(levels))
		}
		f := box(footer)
		if f.X != 24 || f.X+f.W != size.X-24 || f.Y+f.H != size.Y-24 {
			t.Errorf("%v: the footer spans the window inside the margin, at its bottom: %v", size, f)
		}
		above, below := box(levels).Y-(box(title).Y+box(title).H), f.Y-(box(levels).Y+box(levels).H)
		if d := above - below; d < -1 || d > 1 {
			t.Errorf("%v: the spacers share the height alike: %v above, %v below", size, above, below)
		}
	}
}

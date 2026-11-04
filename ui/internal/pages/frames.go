package pages

import (
	"time"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// WorldFrames are a page's whole frames in a headless World: input,
// styling, layout, paint and the draw list the GPU would take, without
// the GPU; the first one, warming fonts and glyphs, a static one, and
// ones where the pointer moves over a button or a text field's value
// changes; and the static frame's draw batches and vertices.
type WorldFrames struct {
	First    Frame  `json:"first"`
	Static   Frame  `json:"static"`
	Hover    *Frame `json:"hover,omitempty"`
	Edit     *Frame `json:"edit,omitempty"`
	Batches  int    `json:"batches"`
	Vertices int    `json:"vertices"`
}

// worldFrames measures d's whole frames in a w×h window.
func worldFrames(d *ui.Document, w, h int) WorldFrames {
	win := platform.NewWindow(platform.Size(w, h))
	win.Add(d)
	world := ember.New().AddEntities(win)
	var f WorldFrames
	start := time.Now()
	world.RunFrames(1)
	f.First = Frame{NsPerOp: time.Since(start).Nanoseconds()}
	world.RunFrames(2)
	f.Static = measure(func() { world.RunFrames(1) })
	if l := draw.FrontFor(world, win); l != nil {
		f.Batches, f.Vertices = len(l.Batches), len(l.Vertices)
	}
	if b := first(d, func(n *ui.Node) bool { return n.Tag() == dom.Button }); b != nil {
		over, away := b.Box().Center(), geom.V2(1, 1)
		on := false
		r := measure(func() {
			on = !on
			at := away
			if on {
				at = over
			}
			win.Input().SetMouse(at)
			world.RunFrames(1)
		})
		f.Hover = &r
	}
	if in := first(d, func(n *ui.Node) bool {
		return n.Tag() == dom.Textarea || n.Tag() == dom.Input &&
			isText(n)
	}); in != nil {
		text := []string{"a", "ab"}
		i := 0
		r := measure(func() {
			i++
			d.SetValue(in, text[i%2])
			world.RunFrames(1)
		})
		f.Edit = &r
	}
	return f
}

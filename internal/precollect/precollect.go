// Package precollect runs what engine packages need done after a frame's
// systems have run and before twod collects what to draw: ui lays out the
// elements shown since its own pass.
package precollect

import (
	"github.com/lantervale/ember/internal/slot"
	"github.com/lantervale/ember/world"
)

type hooks struct{ fns []func(w *world.World) }

// Add has fn run in w before each frame is collected.
func Add(w *world.World, fn func(w *world.World)) {
	h := slot.Get[hooks](w)
	if h == nil {
		h = &hooks{}
		slot.Set(w, h)
	}
	h.fns = append(h.fns, fn)
}

// Run runs w's hooks, in the order they were added.
func Run(w *world.World) {
	if h := slot.Get[hooks](w); h != nil {
		for _, fn := range h.fns {
			fn(w)
		}
	}
}

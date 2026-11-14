// Package pointer lets the ui say which of what is drawn takes the
// pointer, so a covered element neither hovers nor presses; package twod
// asks it.
package pointer

import (
	"github.com/lantervale/ember/internal/slot"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/world"
)

// Gate is a World's rule of what takes the pointer.
type Gate struct{ allow func(node.Noded) bool }

// Set makes allow the rule of w: what it refuses is under something else.
func Set(w *world.World, allow func(node.Noded) bool) { slot.Set(w, &Gate{allow}) }

// Allowed reports whether e, being drawn, takes the pointer in w.
func Allowed(w *world.World, e node.Noded) bool {
	if w == nil || e == nil {
		return true
	}
	g := slot.Get[Gate](w)
	return g == nil || g.allow == nil || g.allow(e)
}

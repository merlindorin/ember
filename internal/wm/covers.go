package wm

import (
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

type cover struct {
	e     Entity
	frame uint64
}

// Cover says e covers win under the pointer this frame (decision 0055):
// a UI layer drawn over others, such as a ui Document with a menu open
// there. Through the next frame only e and what is under it take win's
// pointer; the keys stay everyone's, and win's owner, if it has one,
// still wins. A layer covers again each frame it is under the pointer.
func Cover(w *world.World, win *platform.Window, e Entity) {
	r := of(w)
	if r.covers == nil {
		r.covers = map[*platform.Window]cover{}
	}
	r.covers[win] = cover{e, w.Time().Frame}
}

// Covering is what covered win under the pointer in the frame before
// this one, nil for nothing. A layer covering again this frame stamps its
// cover with this frame, so what is drawn after it, over it, is never
// covered by it.
func Covering(w *world.World, win *platform.Window) Entity {
	c, ok := of(w).covers[win]
	if !ok || c.frame+1 != w.Time().Frame {
		return nil
	}
	return c.e
}

// TakesPointer reports whether e takes win's pointer: it takes win's input
// (Takes), and nothing else covered win under the pointer last frame. A
// cover outside win's owner counts for nothing.
func TakesPointer(w *world.World, win *platform.Window, e Entity) bool {
	if !Takes(w, win, e) {
		return false
	}
	c := Covering(w, win)
	if o := Owner(w, win); c == nil || o != nil && !under(c, o) {
		return true
	}
	return under(e, c)
}

// Uncover drops e's cover of win at once, as a modal closing hands the
// pointer back the same frame.
func Uncover(w *world.World, win *platform.Window, e Entity) {
	r := of(w)
	if c, ok := r.covers[win]; ok && c.e == e {
		delete(r.covers, win)
	}
}

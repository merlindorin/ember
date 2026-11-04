package wm

import (
	"slices"

	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
)

// Entity is what owns a window's input: the entity being drawn when a
// modal opened, a ui document or the element hosting it.
type Entity = interface{ GetBasicEntity() *world.BasicEntity }

// Own puts owner on top of win's owners (decision 0053): until it
// releases it, only owner and what is under it take win's pointer and
// keys among the UI layers, and the window's input reads as captured, so
// game systems stand aside too. The last to own wins; owning again moves
// owner to the top. An owner that leaves the World, or is hidden, it or a
// parent, owns nothing from the next frame; one that only stops drawing
// keeps the window until it releases it.
func Own(w *world.World, win *platform.Window, owner Entity) {
	r := of(w)
	if r.owners == nil {
		r.owners = map[*platform.Window][]Entity{}
	}
	s := slices.DeleteFunc(r.owners[win], func(e Entity) bool { return e == owner })
	r.owners[win] = append(s, owner)
	capture(win)
}

// Release takes owner out of win's owners, wherever it stands: the one
// below takes the input back at once.
func Release(w *world.World, win *platform.Window, owner Entity) {
	r := of(w)
	s := slices.DeleteFunc(r.owners[win], func(e Entity) bool { return e == owner })
	if len(s) == 0 {
		delete(r.owners, win)
		return
	}
	r.owners[win] = s
}

// Owner is the owner on top of win's, nil when nothing owns it.
func Owner(w *world.World, win *platform.Window) Entity {
	s := of(w).owners[win]
	if len(s) == 0 {
		return nil
	}
	return s[len(s)-1]
}

// Owns reports whether owner is among win's owners, on top or not.
func Owns(w *world.World, win *platform.Window, owner Entity) bool {
	return slices.Contains(of(w).owners[win], owner)
}

// Takes reports whether e takes win's input: win has no owner, or e is
// its top one or under it.
func Takes(w *world.World, win *platform.Window, e Entity) bool {
	o := Owner(w, win)
	return o == nil || under(e, o)
}

// Holds reports whether root is win's top owner or holds it among its
// descendants: the one root of win that still takes its pointer.
func Holds(w *world.World, win *platform.Window, root Entity) bool {
	o := Owner(w, win)
	return o == nil || under(o, root)
}

// under reports whether e is top or one of its descendants.
func under(e, top Entity) bool {
	t := top.GetBasicEntity()
	for b := e.GetBasicEntity(); b != nil; b = b.Parent() {
		if b == t {
			return true
		}
	}
	return false
}

func capture(win *platform.Window) {
	f := inputfeed.Of(win.Input())
	f.CapturePointer()
	f.CaptureKeyboard()
}

// owned drops the windows closed, their owners and covers, and the owners
// gone from the World or hidden, and captures every window still owned,
// as each frame begins.
func (r *registry) owned(w *world.World) {
	for win := range r.covers {
		if !r.open(win) {
			delete(r.covers, win)
		}
	}
	for win, s := range r.owners {
		if !r.open(win) {
			delete(r.owners, win)
			continue
		}
		s = slices.DeleteFunc(s, func(e Entity) bool { return !shown(w, e) })
		if len(s) == 0 {
			delete(r.owners, win)
			continue
		}
		r.owners[win] = s
		capture(win)
	}
}

func (r *registry) open(win *platform.Window) bool {
	for e := range r.All() {
		if e.GetWindow() == win {
			return true
		}
	}
	return false
}

// shown reports whether e is in the World with neither it nor a parent
// hidden.
func shown(w *world.World, e Entity) bool {
	if _, in := w.Entity(e.GetBasicEntity().ID()); !in {
		return false
	}
	for b := e.GetBasicEntity(); b != nil; b = b.Parent() {
		if p, ok := w.Entity(b.ID()); ok {
			if h, hides := p.(interface{ Hidden() bool }); hides && h.Hidden() {
				return false
			}
		}
	}
	return true
}

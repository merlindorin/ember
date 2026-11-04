package ui

import (
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// owner is where d draws, and whether d owns that window's input while
// a modal dialog of its is open (decision 0053). Hidden, d owns nothing
// (wm drops it); shown again with the modal still open, it owns again.
type owner struct {
	w      *world.World
	win    *platform.Window
	e      wm.Entity
	owning bool
}

// drawnBy notes where d draws: the entity being drawn is d, or the
// element hosting it.
func (d *Document) drawnBy(c *twod.Canvas) {
	w, win, e := c.World(), c.Window(), c.Drawing()
	if w == nil || win == nil || e == nil {
		return
	}
	if d.owner.owning && (d.owner.win != win || d.owner.e != e) {
		d.release()
	}
	d.owner.w, d.owner.win, d.owner.e = w, win, e
	d.own()
}

// own owns or releases d's window as its modals open and close.
func (d *Document) own() {
	o := &d.owner
	if o.w == nil {
		return
	}
	switch modal := len(d.modals) > 0; {
	case modal && !wm.Owns(o.w, o.win, o.e):
		wm.Own(o.w, o.win, o.e)
		o.owning = true
	case !modal && o.owning:
		d.release()
	}
}

func (d *Document) release() {
	wm.Release(d.owner.w, d.owner.win, d.owner.e)
	wm.Uncover(d.owner.w, d.owner.win, d.owner.e)
	d.owner.owning = false
}

// cover says d covers its window under the pointer this frame, over what
// was drawn before it (decision 0055): one of its elements is there.
func (d *Document) cover(c *twod.Canvas) {
	if w, win, e := c.World(), c.Window(), c.Drawing(); w != nil && win != nil && e != nil {
		wm.Cover(w, win, e)
	}
}

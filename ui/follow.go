package ui

import (
	"sync/atomic"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// following is what a node follows: its entity, the offset, and where it
// was last put, in CSS px of its containing block.
type following struct {
	target    twod.Entity
	offset    geom.Vec2
	left, top float32
	placed    bool
	hidden    bool
}

// Follow keeps the element on a 2D entity's screen point, as the old ember/ui's
// Follow did: each frame, with position: absolute, its box's bottom
// centre sits on the point where the entity draws in the Document's
// window, moved by offset in CSS px, such as a name plate over a head.
// While the entity's point is off the Document's screen, or the entity
// isn't drawn, the element is visibility: hidden. Its first frame
// measures it still in flow (a block as wide as its container), so it
// settles on its second frame once absolute positioning shrinks it.
// 3D entities come with the 3D API.
func Follow(target twod.Entity, offset geom.Vec2) Arg {
	return option(func(n *Node) { SetFollow(n, target, offset) })
}

// SetFollow makes n follow target, moved by offset; a nil target stops it
// where it was, shown.
func SetFollow(n *Node, target twod.Entity, offset geom.Vec2) {
	if target == nil {
		if f, _ := dom.Data(n).Follow.(*following); f != nil && f.hidden {
			setStyleVar(n, "visibility", "")
		}
		dom.Data(n).Follow = nil
		followings.Add(1)
		return
	}
	if f, _ := dom.Data(n).Follow.(*following); f != nil {
		f.target, f.offset = target, offset
		return
	}
	dom.Data(n).Follow = &following{target: target, offset: offset}
	followings.Add(1)
}

// followings counts the elements that started or stopped following, so a
// Document looks for its followers again after one did. It is global
// because SetFollow is given only the node, which may be in no Document
// yet; a bump costs each Document one look.
var followings atomic.Uint64 //nolint:gochecknoglobals // shared by every Document, see above

// findFollowers lists the placed nodes following an entity again, once one
// started or stopped following since d last did.
func (d *Document) findFollowers() {
	seen := followings.Load()
	if seen == d.followingsSeen || d.tree == nil {
		return
	}
	d.followingsSeen, d.followers = seen, d.followers[:0]
	for n := range d.tree.Of {
		if dom.Data(n).Follow != nil {
			d.followers = append(d.followers, n)
		}
	}
}

// follow puts each follower on its entity's screen point, writing its
// place only when it moves, so a still scene stays a quiet frame; it
// reports whether one moved, which needs a layout.
func (d *Document) follow(c *twod.Canvas) bool {
	w := c.World()
	d.findFollowers()
	if w == nil || len(d.followers) == 0 {
		return false
	}
	screen, zoom := c.Screen(), d.Zoom()
	moved := false
	for _, n := range d.followers {
		f, _ := dom.Data(n).Follow.(*following)
		b := d.tree.Of[n]
		if f == nil || b == nil {
			continue
		}
		at, ok := draw.ToScreen(w, f.target)
		ok = ok && at.X >= screen.X && at.Y >= screen.Y && at.X <= screen.X+screen.W && at.Y <= screen.Y+screen.H
		if ok != !f.hidden {
			f.hidden = !ok
			setStyleVar(n, "visibility", map[bool]string{true: "", false: "hidden"}[ok])
			moved = true
		}
		if !ok {
			continue
		}
		r, origin := b.Rect(), d.containerOrigin(n)
		left := (at.X-screen.X)/zoom + f.offset.X - r.W/2 - origin.X
		top := (at.Y-screen.Y)/zoom + f.offset.Y - r.H - origin.Y
		if f.placed && abs(left-f.left) < 0.01 && abs(top-f.top) < 0.01 {
			continue
		}
		f.left, f.top, f.placed = left, top, true
		setStyleVar(n, "position", "absolute")
		setStyleVar(n, "left", px(left))
		setStyleVar(n, "top", px(top))
		moved = true
	}
	return moved
}

// containerOrigin is the padding box's top-left of n's containing block
// for position: absolute, its nearest positioned ancestor, else the
// Document's origin.
func (d *Document) containerOrigin(n *Node) geom.Vec2 {
	for a := n.Parent(); a != nil; a = a.Parent() {
		st := dom.Data(a).Style
		if st == nil || st.Position == style.Static {
			continue
		}
		if b := d.tree.Of[a]; b != nil {
			r := b.Rect()
			return geom.V2(r.X+b.Border.Left, r.Y+b.Border.Top)
		}
	}
	return geom.Vec2{}
}

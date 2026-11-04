package ui

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// BoxOf is n's content box in its window's points (the Document's, times its zoom), for a
// Document drawn in a window, where a game places what it draws itself
// over an element that only reserves room; false while n isn't laid out
// or isn't shown. It is the last layout's: a resize reaches it the frame
// after.
func (d *Document) BoxOf(n *Node) (geom.Rect, bool) {
	if d.tree == nil || n == nil {
		return geom.Rect{}, false
	}
	b := d.tree.Of[n]
	if b == nil {
		return geom.Rect{}, false
	}
	for p := n; p != nil; p = p.Parent() {
		if st := dom.Data(p).Style; st != nil && st.Display == style.DisplayNone {
			return geom.Rect{}, false
		}
	}
	r := n.Box()
	l, t := b.Border.Left+b.Padding.Left, b.Border.Top+b.Padding.Top
	w := r.W - l - b.Border.Right - b.Padding.Right
	h := r.H - t - b.Border.Bottom - b.Padding.Bottom
	return d.toWindow(geom.Rect{X: r.X + l, Y: r.Y + t, W: max(w, 0), H: max(h, 0)}), true
}

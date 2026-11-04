package ui

import (
	"strconv"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// A window is a dialog drawn in the game window holding a bar, an element
// with data-window-bar (a title bar, or a toolbar standing for it), such as
// kit's Window. Its bar's buttons with data-window-action close it
// ("close", through Close), fold it to its bar ("minimise", toggling
// data-minimised) or make it fill the window ("zoom", toggling data-zoomed,
// which clears data-minimised); the button's Clicked still fires. A press on
// the bar, off its controls, drags it, writing left and top into its style
// and marking it data-placed (and data-moved once it moves), so its bar
// stays at least its own height inside the window; with data-resizable, its
// right and bottom edges resize it, within its min-width and min-height. A
// zoomed window does neither. A press in a window shown not modal raises it
// over the others by its z-index.

// grip is the room along a window's right and bottom edges that resizes it.
const grip = 6

type windowMode uint8

const (
	moveWindow windowMode = iota
	resizeRight
	resizeBottom
	resizeCorner
)

// windowDrag is a window held by its bar or an edge: the pointer at the press and the
// window's box then.
type windowDrag struct {
	win   *Node
	mode  windowMode
	from  geom.Vec2
	start geom.Rect
	// to is where the drag has put the window's top-left, or its size.
	to geom.Vec2
}

func barOf(w *Node) *Node {
	for _, c := range w.Children() {
		if has(c, "data-window-bar") {
			return c
		}
	}
	return nil
}

// windowOf is the window n is in, or is, nil for none.
func windowOf(n *Node) *Node {
	for p := n; p != nil; p = p.Parent() {
		if p.Tag() == dom.Dialog && barOf(p) != nil {
			return p
		}
	}
	return nil
}

// windowAction is a click on a window's light: close, minimise or zoom.
func (d *Document) windowAction(n *Node) {
	action, ok := n.Attr("data-window-action")
	w := windowOf(n)
	if !ok || w == nil {
		return
	}
	switch action {
	case "close":
		d.Close(w, "")
	case "minimise":
		if has(w, "data-minimised") {
			w.RemoveAttr("data-minimised")
			d.events = append(d.events, WindowFolded{w, false})
		} else if !has(w, "data-zoomed") {
			w.SetAttr("data-minimised", "")
			d.events = append(d.events, WindowFolded{w, true})
		}
	case "zoom":
		if has(w, "data-zoomed") {
			w.RemoveAttr("data-zoomed")
		} else {
			if has(w, "data-minimised") {
				w.RemoveAttr("data-minimised")
				d.events = append(d.events, WindowFolded{w, false})
			}
			w.SetAttr("data-zoomed", "")
		}
	}
}

// edgeAt is the edge of the window w that at resizes, false off them.
func edgeAt(w *Node, at geom.Vec2) (windowMode, bool) {
	if !has(w, "data-resizable") || has(w, "data-zoomed") {
		return 0, false
	}
	r := w.Box()
	if !r.Contains(at) {
		return 0, false
	}
	right, bottom := at.X >= r.X+r.W-grip, at.Y >= r.Y+r.H-grip
	switch {
	case right && bottom:
		return resizeCorner, true
	case right:
		return resizeRight, true
	case bottom:
		return resizeBottom, true
	}
	return 0, false
}

// edgeCursorOf is the resize shape of m, false for moving a window.
func edgeCursorOf(m windowMode) (platform.Cursor, bool) {
	switch m {
	case resizeRight:
		return platform.CursorResizeEW, true
	case resizeBottom:
		return platform.CursorResizeNS, true
	case resizeCorner:
		return platform.CursorResizeNWSE, true
	case moveWindow:
		// a move has no edge cursor
	}
	return 0, false
}

// windowPointer raises the window a press lands in, and drags or resizes one held by its
// bar or an edge. It reports whether it took the pointer.
func (d *Document) windowPointer(in *input.State, at geom.Vec2) bool {
	if g := d.held; g != nil {
		if !in.Down(input.MouseLeft) || !d.Open(g.win) {
			d.held = nil
			d.dropped(g)
			return true
		}
		d.dragWindow(g, at)
		return true
	}
	w := windowOf(firstOf(d.hovered))
	if w == nil || !in.Pressed(input.MouseLeft) {
		return false
	}
	if len(d.modals) > 0 && d.scopeOf(w) != len(d.modals) {
		return false
	}
	d.raise(w)
	mode, edge := edgeAt(w, at)
	if !edge && (has(w, "data-zoomed") || !contains(d.hovered, barOf(w)) || d.onControl(barOf(w))) {
		return false
	}
	if !edge {
		mode = moveWindow
	}
	d.held = &windowDrag{win: w, mode: mode, from: at, start: w.Box()}
	d.place(w, w.Box().X, w.Box().Y)
	return true
}

// dropped tells where a drag let go put the window, or the size it took
// within its min and max sizes, when it changed.
func (d *Document) dropped(g *windowDrag) {
	switch {
	case g.mode == moveWindow && g.to != (geom.Vec2{}) && g.to != geom.V2(g.start.X, g.start.Y):
		d.events = append(d.events, WindowMoved{g.win, g.to})
	case g.mode != moveWindow && g.to != (geom.Vec2{}):
		b := g.win.Box()
		if size := geom.V2(b.W, b.H); size != geom.V2(g.start.W, g.start.H) {
			d.events = append(d.events, WindowResized{g.win, size})
		}
	}
}

func firstOf(c []*Node) *Node {
	if len(c) == 0 {
		return nil
	}
	return c[0]
}

// onControl reports whether the pointer is over a control inside bar, such as a light or
// a toolbar's button, which a press clicks rather than drags.
func (d *Document) onControl(bar *Node) bool {
	for _, n := range d.hovered {
		if n == bar {
			return false
		}
		if _, ok := focusable(n); ok || d.machine(n) != nil {
			return true
		}
	}
	return false
}

// raise brings the window w, shown not modal, over the others.
func (d *Document) raise(w *Node) {
	if m, ok := d.machine(w).(*control.Dialog); !ok || m.Modal || d.raised == w {
		return
	}
	d.raisedZ++
	d.raised = w
	setDeclaration(w, "z-index", strconv.Itoa(d.raisedZ))
}

// dragWindow follows the pointer at with the held window.
func (d *Document) dragWindow(g *windowDrag, at geom.Vec2) {
	delta := at.Sub(g.from)
	b := d.tree.Of[g.win]
	if b == nil {
		return
	}
	switch g.mode {
	case moveWindow:
		bar := float32(0)
		if bb := d.tree.Of[barOf(g.win)]; bb != nil {
			bar = bb.Size.Y
		}
		x := min(max(g.start.X+delta.X, bar-g.start.W), d.laidAt.X-bar)
		y := min(max(g.start.Y+delta.Y, 0), d.laidAt.Y-bar)
		if delta != (geom.Vec2{}) {
			d.place(g.win, x, y)
			if !has(g.win, "data-moved") {
				g.win.SetAttr("data-moved", "")
			}
			g.to = geom.V2(x, y)
		}
	case resizeRight, resizeBottom, resizeCorner:
		st := dom.Data(g.win).Style
		var inset geom.Vec2
		if st.BoxSizing != style.BorderBox {
			inset = geom.V2(b.Border.Horizontal()+b.Padding.Horizontal(), b.Border.Vertical()+b.Padding.Vertical())
		}
		size := geom.V2(g.start.W, g.start.H)
		if g.mode != resizeBottom {
			w := max(g.start.W+delta.X-inset.X, st.MinSize.Width.Px)
			setDeclaration(g.win, "width", px(w))
			size.X = w + inset.X
		}
		if g.mode != resizeRight {
			h := max(g.start.H+delta.Y-inset.Y, st.MinSize.Height.Px)
			setDeclaration(g.win, "height", px(h))
			size.Y = h + inset.Y
		}
		g.to = size
	}
}

// place puts the window w's top-left at x, y in the window.
func (d *Document) place(w *Node, x, y float32) {
	if !has(w, "data-placed") {
		w.SetAttr("data-placed", "")
	}
	setDeclaration(w, "left", px(x))
	setDeclaration(w, "top", px(y))
}

func px(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) + "px" }

// setDeclaration sets property to value in n's style attribute, keeping the rest.
func setDeclaration(n *Node, property, value string) {
	var kept []string
	for _, decl := range strings.Split(n.Style(), ";") {
		k, _, ok := strings.Cut(decl, ":")
		if !ok || strings.TrimSpace(k) == property {
			continue
		}
		kept = append(kept, strings.TrimSpace(decl))
	}
	kept = append(kept, property+": "+value)
	if s := strings.Join(kept, "; "); s != n.Style() {
		n.SetStyle(s)
	}
}

// edgeCursor shows a resize shape while a window resizes or over its edges, reporting
// whether it did.
func (d *Document) edgeCursor(c *twod.Canvas, at geom.Vec2) bool {
	if g := d.held; g != nil {
		cur, ok := edgeCursorOf(g.mode)
		if ok {
			c.ShowCursor(cur)
		}
		return ok
	}
	if w := windowOf(firstOf(d.hovered)); w != nil {
		if mode, ok := edgeAt(w, at); ok {
			cur, _ := edgeCursorOf(mode)
			c.ShowCursor(cur)
			return true
		}
	}
	return false
}

// PlaceWindow puts the window w's top-left at at, in the Document's
// points, without events; unlike the user's drag, it leaves MovedByUser
// false.
func (d *Document) PlaceWindow(w *Node, at geom.Vec2) { d.place(w, at.X, at.Y) }

// SizeWindow gives the window w a border box of size, without events; a
// side of 0 is left as it is.
func (d *Document) SizeWindow(w *Node, size geom.Vec2) {
	if size.X > 0 {
		setDeclaration(w, "width", px(size.X))
	}
	if size.Y > 0 {
		setDeclaration(w, "height", px(size.Y))
	}
}

// SetFolded folds the window w to its bar, or unfolds it, without events.
func (d *Document) SetFolded(w *Node, on bool) {
	switch {
	case on && !has(w, "data-minimised"):
		w.SetAttr("data-minimised", "")
	case !on && has(w, "data-minimised"):
		w.RemoveAttr("data-minimised")
	}
}

// MovedByUser reports whether the user has dragged the window w by its bar
// (data-moved), as opposed to placing it by code.
func (d *Document) MovedByUser(w *Node) bool { return has(w, "data-moved") }

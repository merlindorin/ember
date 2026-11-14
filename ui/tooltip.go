package ui

import (
	"slices"
	"strings"

	"github.com/lantervale/ember/geom"
)

// A tooltip (role tooltip) is placed once each time it shows: when it
// would cross the window's edge on its side, it moves to the other side
// (data-side left, right, above or below), and when it crosses there too
// it is held inside the window, 4 points from its edges (data-clamp with
// --tip-x and --tip-y, its offset in its anchor's padding box). It keeps
// its place until it hides, which takes the attributes off. Its anchor is
// its parent, the containing block kit's .has-tooltip gives it; kit's
// stylesheet reads the attributes.

// tipGap is the space between a tip and its anchor, kit's .tooltip margin;
// tipMargin is how far a tip held inside the window keeps from its edges.
const (
	tipGap    = 8
	tipMargin = 4
)

func isTip(n *Node) bool {
	r, _ := n.Attr("role")
	return r == "tooltip"
}

// placeTips places the tips that showed since the last layout and frees
// those that hid, and reports whether a tip moved, which needs a layout.
func (d *Document) placeTips() bool {
	for t := range d.placedTips {
		if !slices.Contains(d.tips, t) {
			delete(d.placedTips, t)
			unplace(t)
		}
	}
	moved := false
	for _, t := range d.tips {
		if d.placedTips[t] {
			continue
		}
		if d.placedTips == nil {
			d.placedTips = map[*Node]bool{}
		}
		d.placedTips[t] = true
		moved = d.placeTip(t) || moved
	}
	return moved
}

// placeTip flips or holds the tip t inside the window, reporting whether
// it set an attribute.
func (d *Document) placeTip(t *Node) bool {
	anchor := t.Parent()
	tb, ab := d.tree.Of[t], d.tree.Of[anchor]
	if anchor == nil || tb == nil || ab == nil {
		return false
	}
	r, a := tb.Rect(), ab.Rect()
	if inside(r, geom.Rect{W: d.laidAt.X, H: d.laidAt.Y}) {
		return false
	}
	view := geom.Rect{X: tipMargin, Y: tipMargin, W: d.laidAt.X - 2*tipMargin, H: d.laidAt.Y - 2*tipMargin}
	side := sideRight
	switch {
	case r.X+r.W <= a.X:
		side = sideLeft
	case r.Y >= a.Y+a.H:
		side = "below"
	}
	at := func(side string) geom.Rect {
		switch side {
		case sideLeft:
			return geom.Rect{X: a.X - tipGap - r.W, Y: r.Y, W: r.W, H: r.H}
		case sideRight:
			return geom.Rect{X: a.X + a.W + tipGap, Y: r.Y, W: r.W, H: r.H}
		case sideAbove:
			return geom.Rect{X: r.X, Y: a.Y - tipGap - r.H, W: r.W, H: r.H}
		}
		return geom.Rect{X: r.X, Y: a.Y + a.H + tipGap, W: r.W, H: r.H}
	}
	other := map[string]string{sideRight: sideLeft, sideLeft: sideRight, "below": sideAbove}[side]
	if flipped := at(other); inside(flipped, view) {
		t.SetAttr("data-side", other)
		return true
	}
	x := min(max(r.X, view.X), view.X+view.W-r.W)
	y := min(max(r.Y, view.Y), view.Y+view.H-r.H)
	origin := geom.V2(a.X+ab.Border.Left, a.Y+ab.Border.Top)
	t.SetAttr("data-clamp", "")
	setStyleVar(t, "--tip-x", px(x-origin.X))
	setStyleVar(t, "--tip-y", px(y-origin.Y))
	return true
}

// unplace takes off what placeTip set on t.
func unplace(t *Node) {
	t.RemoveAttr("data-side")
	t.RemoveAttr("data-clamp")
	setStyleVar(t, "--tip-x", "")
	setStyleVar(t, "--tip-y", "")
}

func inside(r, view geom.Rect) bool {
	return r.X >= view.X-0.5 && r.Y >= view.Y-0.5 && r.X+r.W <= view.X+view.W+0.5 && r.Y+r.H <= view.Y+view.H+0.5
}

// setStyleVar sets the declaration name: value in n's style attribute,
// keeping its others; an empty value takes it out.
func setStyleVar(n *Node, name, value string) {
	var kept []string
	for _, d := range strings.Split(n.Style(), ";") {
		if d = strings.TrimSpace(d); d != "" && !strings.HasPrefix(d, name+":") {
			kept = append(kept, d)
		}
	}
	if value != "" {
		kept = append(kept, name+": "+value)
	}
	if s := strings.Join(kept, "; "); s != n.Style() {
		n.SetStyle(s)
	}
}

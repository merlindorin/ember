package ui

import (
	"math"

	"github.com/merlindorin/ember/ui/control"
)

// A listbox is WAI-ARIA's listbox pattern, its options laid out in lines as CSS places
// them, such as kit's Thumbnails: one Tab stop whose options (role option) the player
// selects, the selected one with aria-selected="true", those with aria-disabled="true"
// shown but skipped. A click or the arrows, Home and End select an option, emitting
// Changed for the listbox: Left and Right step through them in order, Up and Down to the
// nearest on the line above or below. Enter, Space, the pad's south button or a double
// click activate the selection, emitting Submitted for it. The selection scrolls into
// view.

func isListbox(n *Node) bool { return roleOf(n) == "listbox" }

// listboxOf is the listbox n is in, or is, nil for none.
func listboxOf(n *Node) *Node {
	for p := n; p != nil; p = p.Parent() {
		if isListbox(p) {
			return p
		}
	}
	return nil
}

// isOption reports whether n is a listbox's option.
func isOption(n *Node) bool { return roleOf(n) == roleOption && listboxOf(n.Parent()) != nil }

// listOptions are listbox's options, in tree order.
func listOptions(listbox *Node) []*Node {
	var out []*Node
	listbox.Walk(func(n *Node) bool {
		if n.IsText() || n != listbox && isListbox(n) {
			return false
		}
		if roleOf(n) == roleOption {
			out = append(out, n)
			return false
		}
		return true
	})
	return out
}

// selectOption selects o among its listbox's options, reporting whether that changed
// anything.
func selectOption(listbox, o *Node) bool {
	if ariaDisabled(o) || ariaSelected(o) {
		return false
	}
	for _, x := range listOptions(listbox) {
		if x != o && ariaSelected(x) {
			x.SetAttr("aria-selected", "false")
		}
	}
	o.SetAttr("aria-selected", attrTrue)
	return true
}

// selectedOption is the place of listbox's selected option among its options, -1 for
// none.
func selectedOption(listbox *Node) int {
	for i, o := range listOptions(listbox) {
		if ariaSelected(o) {
			return i
		}
	}
	return -1
}

// listboxClick is a click on the option n: it is selected, then activated by a second
// click on it soon after. It reports whether n was an option.
func (d *Document) listboxClick(n *Node) bool {
	if !isOption(n) {
		return false
	}
	lb := listboxOf(n.Parent())
	if selectOption(lb, n) {
		d.fire(lb, control.Changed)
	} else if n == d.lastPress && d.clock-d.lastPressAt <= doubleClick && !ariaDisabled(n) {
		d.fire(lb, control.Submitted)
		d.lastPress = nil
		return true
	}
	d.lastPress, d.lastPressAt = n, d.clock
	return true
}

// listboxIntent moves listbox's selection for i, or activates it; false for intents a
// listbox doesn't take.
func (d *Document) listboxIntent(listbox *Node, i control.Intent) bool {
	var live []*Node
	at := -1
	for _, o := range listOptions(listbox) {
		if !ariaDisabled(o) {
			if ariaSelected(o) {
				at = len(live)
			}
			live = append(live, o)
		}
	}
	switch i {
	case control.Activate:
		if at >= 0 {
			d.fire(listbox, control.Submitted)
		}
		return true
	case control.Left, control.Right, control.Up, control.Down, control.Home, control.End:
	case control.None, control.Back, control.PageUp, control.PageDown:
		return false
	}
	if len(live) == 0 {
		return true
	}
	next := 0
	switch {
	case i == control.Home || at < 0 && i != control.End:
	case i == control.End:
		next = len(live) - 1
	case i == control.Left:
		next = max(at-1, 0)
	case i == control.Right:
		next = min(at+1, len(live)-1)
	default:
		next = d.nearestOnLine(live, at, i == control.Down)
	}
	if selectOption(listbox, live[next]) {
		d.fire(listbox, control.Changed)
	}
	d.reveal(live[next])
	return true
}

// nearestOnLine is the option of live on the line below live[at] (above unless down)
// nearest it across, at itself when there is no such line.
func (d *Document) nearestOnLine(live []*Node, at int, down bool) int {
	from := d.tree.Of[live[at]]
	if from == nil {
		return at
	}
	fr := from.Rect()
	cx := fr.X + fr.W/2
	best, bestLine, bestDX := at, float32(math.Inf(1)), float32(math.Inf(1))
	for j, o := range live {
		b := d.tree.Of[o]
		if b == nil || j == at {
			continue
		}
		r := b.Rect()
		gap := fr.Y - (r.Y + r.H)
		if down {
			gap = r.Y - (fr.Y + fr.H)
		}
		if gap < -0.5 {
			continue
		}
		dx := float32(math.Abs(float64(r.X + r.W/2 - cx)))
		if gap < bestLine-0.5 || gap <= bestLine+0.5 && dx < bestDX {
			best, bestLine, bestDX = j, gap, dx
		}
	}
	return best
}

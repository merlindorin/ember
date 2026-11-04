package ui

import (
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// A radio is an input of type radio, grouped with the others of its name
// in the document, or an element of role radio, grouped by its nearest
// role=radiogroup ancestor and checked by aria-checked="true" (WAI-ARIA's
// radio group). Either way a click or Activate checks it and unchecks the
// rest of its group, the arrows move the check along the group, wrapping,
// and only the group's checked one (else its first) is in the Tab order.

func isAriaRadio(n *Node) bool { r, _ := n.Attr("role"); return r == "radio" }

// isRadio reports whether n is a radio of either kind.
func (d *Document) isRadio(n *Node) bool {
	if n == nil || n.IsText() {
		return false
	}
	_, native := d.machine(n).(*control.Radio)
	return native || isAriaRadio(n)
}

// radioGroup is the radios n is checked among, in tree order.
func (d *Document) radioGroup(n *Node) []*Node {
	if isAriaRadio(n) {
		scope := n.Parent()
		for ; scope != nil; scope = scope.Parent() {
			if r, _ := scope.Attr("role"); r == "radiogroup" {
				break
			}
		}
		if scope == nil {
			return []*Node{n}
		}
		var out []*Node
		scope.Walk(func(c *Node) bool {
			if !c.IsText() && isAriaRadio(c) {
				out = append(out, c)
			}
			return !c.IsText()
		})
		return out
	}
	name, _ := n.Attr("name")
	if name == "" {
		return []*Node{n}
	}
	var out []*Node
	d.html.Walk(func(c *Node) bool {
		if c.IsText() {
			return false
		}
		if _, ok := d.machine(c).(*control.Radio); ok {
			if cn, _ := c.Attr("name"); cn == name {
				out = append(out, c)
			}
		}
		return true
	})
	return out
}

// radioChecked reports whether the radio n is checked.
func (d *Document) radioChecked(n *Node) bool {
	if r, ok := d.machine(n).(*control.Radio); ok {
		return r.Checked
	}
	v, _ := n.Attr("aria-checked")
	return v == attrTrue
}

// setRadio checks or unchecks the radio n, without events.
func (d *Document) setRadio(n *Node, on bool) {
	if r, ok := d.machine(n).(*control.Radio); ok {
		r.Checked = on
		d.sync(n, r)
		return
	}
	if on {
		n.SetAttr("aria-checked", attrTrue)
	} else {
		n.SetAttr("aria-checked", "false")
	}
}

// checkRadio checks n and unchecks the rest of its group, emitting Edited
// and Changed when it wasn't checked.
func (d *Document) checkRadio(n *Node) {
	if has(n, "disabled") || d.radioChecked(n) {
		return
	}
	for _, o := range d.radioGroup(n) {
		if o != n {
			d.setRadio(o, false)
		}
	}
	d.setRadio(n, true)
	d.fire(n, control.Input|control.Changed)
}

// radioArrow moves the check from the focused radio n along its group for
// i, an arrow, focusing the radio it lands on; false for other intents.
func (d *Document) radioArrow(n *Node, i control.Intent) bool {
	var step int
	switch i {
	case control.Down, control.Right:
		step = 1
	case control.Up, control.Left:
		step = -1
	case control.None, control.Activate, control.Back, control.Home, control.End, control.PageUp, control.PageDown:
		return false
	}
	group := d.radioGroup(n)
	at := -1
	for k, o := range group {
		if o == n {
			at = k
		}
	}
	if at < 0 || len(group) < 2 {
		return true
	}
	for k := 1; k < len(group); k++ {
		next := group[(at+step*k+len(group)*k)%len(group)]
		if has(next, "disabled") || next.Box().W == 0 {
			continue
		}
		d.moveFocus(func() bool { return d.focus.Set(next, true) })
		d.checkRadio(next)
		return true
	}
	return true
}

// radioTabIndex is the tabindex a radio takes in the Tab order: -1 (click
// and arrows only) unless it is its group's checked one, or its first
// when none is.
func (d *Document) radioTabIndex(n *Node, i int) int {
	group := d.radioGroup(n)
	for _, o := range group {
		if d.radioChecked(o) {
			if o == n {
				return i
			}
			return -1
		}
	}
	for _, o := range group {
		if !has(o, "disabled") {
			if o == n {
				return i
			}
			return -1
		}
	}
	return i
}

func radioState(n *Node, on bool) { n.SetState(dom.Checked, on) }

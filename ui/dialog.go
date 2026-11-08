package ui

import (
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// Closed is a dialog closed: by Close, or by Escape or the pad's east
// button while modal. Value is its return value, empty for Escape.
type Closed struct {
	Node  *Node
	Value string
}

// Show opens the dialog n, not modal: the page stays live around it. As
// HTML's show(), the focus moves to its element with autofocus, if any.
func (d *Document) Show(n *Node) {
	if m, ok := d.machine(n).(*control.Dialog); ok {
		m.Show()
		d.focusing = n
		d.sync(n, m)
	}
}

// ShowModal opens the dialog n modal, as HTML's showModal: above the
// page, which a backdrop covers and which takes neither the pointer nor
// the focus, the focus moving to the dialog's element with autofocus,
// else its first control, until it closes. A button with
// formmethod="dialog" closes it with its value.
func (d *Document) ShowModal(n *Node) {
	m, ok := d.machine(n).(*control.Dialog)
	if !ok || m.Open && m.Modal {
		return
	}
	m.ShowModal()
	d.modals = append(d.modals, n)
	d.own()
	d.focusing = n
	d.opening = true
	d.sync(n, m)
}

// Close closes the dialog n with value as its return value, emitting
// Closed; the focus goes back where it was before a modal opened.
func (d *Document) Close(n *Node, value string) {
	if m, ok := d.machine(n).(*control.Dialog); ok {
		d.fire(n, m.Close(value))
		d.closed(n, m)
	}
}

// Open reports whether the dialog or details n is open.
func (d *Document) Open(n *Node) bool {
	switch m := d.machine(n).(type) {
	case *control.Dialog:
		return m.Open
	case *control.Details:
		return m.Open
	}
	return false
}

// SetOpen opens or closes the details n, without events; it does nothing
// to anything else (a dialog opens by Show or ShowModal).
func (d *Document) SetOpen(n *Node, on bool) {
	if m, ok := d.machine(n).(*control.Details); ok {
		m.SetOpen(on)
		d.sync(n, m)
	}
}

// closed is the dialog n closed, by Close or Back: its focus trap ends.
func (d *Document) closed(n *Node, m *control.Dialog) {
	for i, x := range d.modals {
		if x == n {
			d.modals = append(d.modals[:i], d.modals[i+1:]...)
			d.focus.Release()
			d.own()
			break
		}
	}
	d.sync(n, m)
}

// scopeOf is the modal scope n is in: the place, from 1, of the innermost
// open modal dialog holding it, 0 for the page.
func (d *Document) scopeOf(n *Node) int {
	for p := n; p != nil; p = p.Parent() {
		if p.Tag() != dom.Dialog {
			continue
		}
		for i, m := range d.modals {
			if m == p {
				return i + 1
			}
		}
	}
	return 0
}

// trap moves the focus into the dialog just shown once its controls are
// among the targets, the frame after it opened, trapping it there when
// the dialog is modal: to its autofocus, else its first control, else,
// modal, the dialog itself, as Chrome does.
func (d *Document) trap() {
	n := d.focusing
	d.focusing = nil
	if n == nil || !d.Open(n) {
		return
	}
	if s := d.scopeOf(n); s > 0 {
		d.focus.Trap(s)
	}
	var first *Node
	n.Walk(func(n *Node) bool {
		if first == nil && !n.IsText() && has(n, "autofocus") {
			first = n
		}
		return first == nil
	})
	if first != nil {
		d.focus.Set(first, true)
	}
	if d.scopeOf(n) > 0 && !holds(n, d.focused()) {
		d.focus.Set(n, true)
	}
}

// dialogClick is a click on n: a button with formmethod="dialog" closes
// its dialog with the button's value as the return value, as a dialog
// form's submit button does in HTML.
func (d *Document) dialogClick(n *Node) {
	if m, _ := n.Attr("formmethod"); m != "dialog" {
		return
	}
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Tag() == dom.Dialog {
			v, _ := n.Attr("value")
			d.Close(p, v)
			return
		}
	}
}

// backFromModal closes the newest modal dialog on Back, reporting whether
// one took it.
func (d *Document) backFromModal() bool {
	if len(d.modals) == 0 {
		return false
	}
	n := d.modals[len(d.modals)-1]
	m, ok := d.machine(n).(*control.Dialog)
	if !ok || !m.Handles(control.Back) {
		return false
	}
	d.fire(n, m.Do(control.Back))
	d.closed(n, m)
	return true
}

// holds reports whether x is n or under it.
func holds(n, x *Node) bool {
	for ; x != nil; x = x.Parent() {
		if x == n {
			return true
		}
	}
	return false
}

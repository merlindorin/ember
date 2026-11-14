package ui

import (
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
)

// A reset button is a button of type reset whose aria-controls names a
// text field, or without one sits beside it: a click empties that field, as HTML's reset brings a form's
// fields back, and the field emits Edited and Changed; the focus stays
// where it was. One that controls a search field shows only while the
// search holds text, as a search field's clear button.

// resetField is the text field the reset button n empties, nil when n
// isn't one.
func (d *Document) resetField(n *Node) *Node {
	if n == nil || n.IsText() || n.Tag() != dom.Button {
		return nil
	}
	if t, _ := n.Attr("type"); t != "reset" {
		return nil
	}
	if id, _ := n.Attr("aria-controls"); id != "" {
		if f := d.byID(id); f != nil && textOf(d.machine(f)) != nil {
			return f
		}
		return nil
	}
	if p := n.Parent(); p != nil {
		for _, f := range p.Children() {
			if !f.IsText() && textOf(d.machine(f)) != nil {
				return f
			}
		}
	}
	return nil
}

// reset empties the field the reset button n controls.
func (d *Document) reset(n *Node) {
	f := d.resetField(n)
	t := textOf(d.machine(f))
	if t == nil || t.Empty() {
		return
	}
	t.SetValue("")
	d.fire(f, control.Input|control.Changed)
}

// showReset hides the reset button n while the search it clears is empty.
func (d *Document) showReset(n *Node) {
	f := d.resetField(n)
	t := textOf(d.machine(f))
	if t == nil || !t.Search() {
		return
	}
	switch _, hidden := n.Attr("hidden"); {
	case t.Empty() && !hidden:
		n.SetAttr("hidden", "")
	case !t.Empty() && hidden:
		n.RemoveAttr("hidden")
	}
}

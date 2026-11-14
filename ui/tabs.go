package ui

import "github.com/lantervale/ember/ui/control"

// Tabs follow WAI-ARIA's tabs pattern: a role=tablist holds role=tab
// elements, the one with aria-selected="true" selected, each naming its
// role=tabpanel by aria-controls. Only the selected tab's panel shows (the
// others take the hidden attribute); a click selects a tab; the arrows
// move along the tablist, wrapping, Home and End to its ends, selecting
// each tab they focus (automatic activation), past disabled ones; only the
// selected tab is in the Tab order. A change emits Changed for the
// tablist.

func isTab(n *Node) bool { return roleOf(n) == "tab" }

// tablistOf is the tablist the tab n is in, nil for none.
func tablistOf(n *Node) *Node {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if roleOf(p) == roleTablist {
			return p
		}
	}
	return nil
}

// tabsOf is the tablist's tabs, in tree order, those of tablists inside
// it left out.
func tabsOf(list *Node) []*Node {
	var out []*Node
	for _, c := range list.Children() {
		c.Walk(func(n *Node) bool {
			if n.IsText() || roleOf(n) == roleTablist {
				return false
			}
			if isTab(n) {
				out = append(out, n)
			}
			return true
		})
	}
	return out
}

// siblingTabs are the tabs of n's tablist, n alone without one.
func siblingTabs(n *Node) []*Node {
	if list := tablistOf(n); list != nil {
		return tabsOf(list)
	}
	return []*Node{n}
}

func tabSelected(n *Node) bool { v, _ := n.Attr("aria-selected"); return v == attrTrue }

// selectTab selects the tab n among its tablist's, showing its panel,
// and reports whether that changed anything.
func (d *Document) selectTab(n *Node) bool {
	if has(n, "disabled") || tabSelected(n) {
		return false
	}
	n.SetAttr("aria-selected", attrTrue)
	for _, t := range siblingTabs(n) {
		if t != n {
			t.SetAttr("aria-selected", "false")
		}
		d.showPanel(t)
	}
	return true
}

// chooseTab selects n as the player does, emitting Changed for its
// tablist.
func (d *Document) chooseTab(n *Node) {
	if d.selectTab(n) {
		if list := tablistOf(n); list != nil {
			d.fire(list, control.Changed)
		}
	}
}

// showPanel shows or hides the panel of tab n as n is selected.
func (d *Document) showPanel(n *Node) {
	id, _ := n.Attr("aria-controls")
	if id == "" {
		return
	}
	p := d.byID(id)
	if p == nil {
		return
	}
	hidden := has(p, "hidden")
	if on := tabSelected(n); on && hidden {
		p.RemoveAttr("hidden")
	} else if !on && !hidden {
		p.SetAttr("hidden", "")
	}
}

// tabArrow moves from the focused tab n for i, selecting the tab it lands
// on; false for intents tabs don't take.
func (d *Document) tabArrow(n *Node, i control.Intent) bool {
	var live []*Node
	for _, t := range siblingTabs(n) {
		if !has(t, "disabled") && t.Box().W > 0 || t == n {
			live = append(live, t)
		}
	}
	var next *Node
	switch i {
	case control.Right, control.Down:
		next = step(live, n, 1)
	case control.Left, control.Up:
		next = step(live, n, -1)
	case control.Home:
		next = live[0]
	case control.End:
		next = live[len(live)-1]
	case control.None, control.Activate, control.Back, control.PageUp, control.PageDown:
		return false
	}
	if next != nil && next != n {
		d.moveFocus(func() bool { return d.focus.Set(next, true) })
		d.chooseTab(next)
	}
	return true
}

// tabTabIndex is the tabindex a tab takes in the Tab order: -1 unless it
// is its tablist's selected one, or its first when none is.
func (d *Document) tabTabIndex(n *Node, i int) int {
	tabs := siblingTabs(n)
	for _, t := range tabs {
		if tabSelected(t) {
			if t == n {
				return i
			}
			return -1
		}
	}
	if len(tabs) > 0 && tabs[0] == n {
		return i
	}
	return -1
}

// tabIndexOf is the place of the selected tab in the tablist list, -1
// for none.
func tabIndexOf(list *Node) int {
	for i, t := range tabsOf(list) {
		if tabSelected(t) {
			return i
		}
	}
	return -1
}

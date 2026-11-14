package ui

import (
	"strings"
	"unicode"

	"github.com/lantervale/ember/ui/control"
)

// Menus follow WAI-ARIA's menu and menubar patterns, keyed on roles: a
// role=menubar or role=menu holds items of role menuitem (menuitemcheckbox
// and menuitemradio count as items too); an item's submenu is its child of
// role menu, or the element its aria-controls names, and a button with
// aria-haspopup="menu" opens one the same way. A menu shows while it has
// the open attribute, which the user-agent sheet needs to display it, and
// its opener's aria-expanded follows. Choosing an item that opens nothing
// emits Clicked for it and closes every menu.

func roleOf(n *Node) string {
	if n == nil || n.IsText() {
		return ""
	}
	r, _ := n.Attr("role")
	return r
}

// isMenuItem reports whether n is a menu item of any kind.
func isMenuItem(n *Node) bool {
	switch roleOf(n) {
	case "menuitem", roleMenuItemCheckbox, roleMenuItemRadio:
		return true
	}
	return false
}

func isMenuBox(n *Node) bool { r := roleOf(n); return r == roleMenu || r == roleMenubar }

// opensMenu reports whether n opens a menu: an item or a menu button.
func opensMenu(n *Node) bool {
	if isMenuItem(n) {
		return true
	}
	p, _ := n.Attr("aria-haspopup")
	return p == roleMenu || p == attrTrue
}

// submenu is the menu n opens, nil for none.
func (d *Document) submenu(n *Node) *Node {
	if n == nil || !opensMenu(n) {
		return nil
	}
	if id, ok := n.Attr("aria-controls"); ok {
		if m := d.byID(id); m != nil && roleOf(m) == roleMenu {
			return m
		}
	}
	for _, c := range n.Children() {
		if roleOf(c) == roleMenu {
			return c
		}
	}
	return nil
}

// menuBoxOf is the menu or menubar holding the item n.
func menuBoxOf(n *Node) *Node {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if isMenuBox(p) {
			return p
		}
	}
	return nil
}

// openerOf is the item or button that opens menu m, nil for a menu that
// stands alone.
func (d *Document) openerOf(m *Node) *Node {
	if p := m.Parent(); p != nil && opensMenu(p) {
		return p
	}
	if m.ID() == "" {
		return nil
	}
	var o *Node
	d.html.Walk(func(n *Node) bool {
		if o == nil && !n.IsText() && opensMenu(n) {
			if id, ok := n.Attr("aria-controls"); ok && id == m.ID() {
				o = n
			}
		}
		return o == nil && !n.IsText()
	})
	return o
}

func menuDisabled(n *Node) bool {
	v, _ := n.Attr("aria-disabled")
	return v == attrTrue || has(n, "disabled")
}

// menuItems is box's own items, in tree order: not those of menus inside
// it, nor disabled or undisplayed ones.
func menuItems(box *Node) []*Node {
	var out []*Node
	var walk func(n *Node)
	walk = func(n *Node) {
		for _, c := range n.Children() {
			if c.IsText() || isMenuBox(c) {
				continue
			}
			if isMenuItem(c) {
				if !menuDisabled(c) && c.Box().W > 0 {
					out = append(out, c)
				}
				continue
			}
			walk(c)
		}
	}
	walk(box)
	return out
}

// menuLabel is an item's aria-label, else its own text: its submenu's left
// out.
func menuLabel(n *Node, text func(*Node) string) string {
	if l, ok := n.Attr("aria-label"); ok {
		return l
	}
	var b strings.Builder
	var walk func(n *Node)
	walk = func(n *Node) {
		for _, c := range n.Children() {
			switch {
			case c.IsText():
				b.WriteString(text(c))
			case roleOf(c) != roleMenu:
				walk(c)
			}
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}

func isOpenMenu(m *Node) bool { return m != nil && has(m, "open") }

// openMenu opens the menu item n opens, closing its siblings' menus, and
// focuses its first item (last when last is set) once it is laid out,
// unless focus is false.
func (d *Document) openMenu(n *Node, focus, last bool) {
	m := d.submenu(n)
	if m == nil {
		return
	}
	if box := menuBoxOf(n); box != nil {
		for _, sib := range menuItems(box) {
			if sib != n {
				d.closeMenu(d.submenu(sib))
			}
		}
	}
	m.SetAttr("open", "")
	n.SetAttr("aria-expanded", attrTrue)
	if focus {
		d.menuFocus, d.menuFocusLast = m, last
	}
}

// closeMenu closes m and every menu open inside it.
func (d *Document) closeMenu(m *Node) {
	if !isOpenMenu(m) {
		return
	}
	for _, it := range menuItems(m) {
		d.closeMenu(d.submenu(it))
	}
	m.RemoveAttr("open")
	if o := d.openerOf(m); o != nil {
		o.SetAttr("aria-expanded", "false")
	}
	if d.menuFocus == m {
		d.menuFocus = nil
	}
}

// menuRoot is the opener at the root of the menus n is in: the menubar
// item or menu button the open chain started from, n itself outside any
// menu.
func (d *Document) menuRoot(n *Node) *Node {
	root := n
	for {
		box := menuBoxOf(root)
		if box == nil || roleOf(box) == roleMenubar {
			return root
		}
		o := d.openerOf(box)
		if o == nil {
			return root
		}
		root = o
	}
}

// closeAll closes the menus open from n's root, and gives the root the
// focus back when n had it.
func (d *Document) closeAll(n *Node, refocus bool) {
	root := d.menuRoot(n)
	if box := menuBoxOf(root); box != nil && roleOf(box) == roleMenubar {
		for _, it := range menuItems(box) {
			d.closeMenu(d.submenu(it))
		}
	} else {
		d.closeMenu(d.submenu(root))
	}
	if refocus {
		d.moveFocus(func() bool { return d.focus.Set(root, d.focus.Active()) })
	}
}

// openMenus is every open menu, outermost first.
func (d *Document) openMenus() []*Node {
	var out []*Node
	d.html.Walk(func(n *Node) bool {
		if roleOf(n) == roleMenu && isOpenMenu(n) {
			out = append(out, n)
		}
		return !n.IsText()
	})
	return out
}

// focusMenuItem focuses item, shown as by keys when keys is set.
func (d *Document) focusMenuItem(item *Node, keys bool) {
	d.moveFocus(func() bool { return d.focus.Set(item, keys) })
}

// menuFocusNow focuses the first (or last) item of the menu opened last
// frame, now that it is laid out.
func (d *Document) menuFocusNow() {
	m := d.menuFocus
	if m == nil {
		return
	}
	items := menuItems(m)
	if len(items) == 0 {
		return
	}
	d.menuFocus = nil
	it := items[0]
	if d.menuFocusLast {
		it = items[len(items)-1]
	}
	d.focusMenuItem(it, true)
}

// step is the item i places from n among items, wrapping.
func step(items []*Node, n *Node, i int) *Node {
	at := -1
	for k, it := range items {
		if it == n {
			at = k
		}
	}
	if at < 0 || len(items) == 0 {
		if len(items) > 0 {
			return items[0]
		}
		return nil
	}
	return items[((at+i)%len(items)+len(items))%len(items)]
}

// menuClick is a click on n, an item or a menu button: it toggles the
// menu n opens, or chooses n; false when n is neither.
func (d *Document) menuClick(n *Node) bool {
	if !opensMenu(n) {
		return false
	}
	if m := d.submenu(n); m != nil {
		if isOpenMenu(m) {
			d.closeMenu(m)
		} else {
			d.openMenu(n, false, false)
		}
		return true
	}
	if !isMenuItem(n) {
		return false
	}
	if menuDisabled(n) {
		return true
	}
	d.choose(n)
	return true
}

// menuIntent handles i on the focused item n as WAI-ARIA's menus do,
// reporting whether it did.
//
//nolint:gocyclo,cyclop,funlen // one case per key of WAI-ARIA's menus
func (d *Document) menuIntent(n *Node, i control.Intent) bool {
	if !isMenuItem(n) {
		if opensMenu(n) && d.submenu(n) != nil {
			switch i {
			case control.Activate, control.Down:
				d.openMenu(n, true, false)
				return true
			case control.Up:
				d.openMenu(n, true, true)
				return true
			case control.None, control.Back, control.Left, control.Right, control.Home, control.End, control.PageUp,
				control.PageDown:
				// the others leave a closed menu to its opener
			}
		}
		return false
	}
	box := menuBoxOf(n)
	if box == nil {
		return false
	}
	items := menuItems(box)
	bar := roleOf(box) == roleMenubar
	sub := d.submenu(n)
	switch {
	case i == control.Home:
		d.focusMenuItem(items[0], true)
	case i == control.End:
		d.focusMenuItem(items[len(items)-1], true)
	case bar && (i == control.Left || i == control.Right):
		dir := sideways(i)
		next := step(items, n, dir)
		wasOpen := isOpenMenu(sub)
		d.closeMenu(sub)
		d.focusMenuItem(next, true)
		if wasOpen {
			d.openMenu(next, true, false)
		}
	case bar && (i == control.Down || i == control.Activate && sub != nil):
		d.openMenu(n, true, false)
	case bar && i == control.Up:
		d.openMenu(n, true, true)
	case !bar && i == control.Down:
		d.focusMenuItem(step(items, n, 1), true)
	case !bar && i == control.Up:
		d.focusMenuItem(step(items, n, -1), true)
	case !bar && i == control.Right && sub != nil:
		d.openMenu(n, true, false)
	case !bar && (i == control.Right || i == control.Left):
		opener := d.openerOf(box)
		if i == control.Left && opener != nil && roleOf(menuBoxOf(opener)) == roleMenu {
			d.closeMenu(box)
			d.focusMenuItem(opener, true)
			return true
		}
		root := d.menuRoot(n)
		if rb := menuBoxOf(root); rb != nil && roleOf(rb) == roleMenubar {
			dir := sideways(i)
			next := step(menuItems(rb), root, dir)
			d.closeAll(n, false)
			d.focusMenuItem(next, true)
			d.openMenu(next, true, false)
		}
	case i == control.Activate && sub != nil:
		d.openMenu(n, true, false)
	case i == control.Activate:
		d.menuClick(n)
	case i == control.Back:
		if bar {
			return false
		}
		d.closeMenu(box)
		if o := d.openerOf(box); o != nil {
			d.focusMenuItem(o, true)
		}
	default:
		return false
	}
	return true
}

// menuType moves the focus within n's menu to the next item whose label
// starts with r, wrapping: WAI-ARIA's typeahead.
func (d *Document) menuType(n *Node, r rune) {
	box := menuBoxOf(n)
	if box == nil || unicode.IsSpace(r) {
		return
	}
	items := menuItems(box)
	r = unicode.ToLower(r)
	for k := 1; k <= len(items); k++ {
		it := step(items, n, k)
		if label := []rune(strings.ToLower(menuLabel(it, d.shown))); len(label) > 0 && label[0] == r {
			d.focusMenuItem(it, true)
			return
		}
	}
}

// menuHover is the pointer over chain: while one of a menubar's menus is
// open, another of its items under the pointer opens its own instead, and
// an item under the pointer in an open menu takes the focus quietly.
func (d *Document) menuHover(chain []*Node) {
	for _, n := range chain {
		if !isMenuItem(n) {
			continue
		}
		box := menuBoxOf(n)
		if box == nil {
			return
		}
		if roleOf(box) == roleMenubar {
			for _, it := range menuItems(box) {
				if it != n && isOpenMenu(d.submenu(it)) && d.submenu(n) != nil {
					d.closeMenu(d.submenu(it))
					d.openMenu(n, false, false)
					d.focusMenuItem(n, false)
				}
			}
			return
		}
		if isOpenMenu(box) && d.focused() != n {
			d.focusMenuItem(n, false)
		}
		return
	}
}

// menuPressOutside closes the open menus when the pointer is pressed on
// nothing of theirs: neither a menu nor the item or button that opens one.
func (d *Document) menuPressOutside(chain []*Node) {
	open := d.openMenus()
	if len(open) == 0 {
		return
	}
	for _, n := range chain {
		if roleOf(n) == roleMenu && isOpenMenu(n) || opensMenu(n) && isOpenMenu(d.submenu(n)) || roleOf(n) == roleMenubar {
			return
		}
	}
	for _, m := range open {
		d.closeMenu(m)
	}
}

// menuTabIndex is the tabindex a menu item takes in the Tab order: a
// menubar's first item, or the one focused, is its one Tab stop; items of
// menus are reached by the arrows only.
func (d *Document) menuTabIndex(n *Node) int {
	box := menuBoxOf(n)
	if box == nil || roleOf(box) != roleMenubar {
		return -1
	}
	items := menuItems(box)
	if f := d.focused(); f != nil && menuBoxOf(f) == box {
		if f == n {
			return 0
		}
		return -1
	}
	if len(items) > 0 && items[0] == n {
		return 0
	}
	return -1
}

// sideways is -1 for Left and 1 for Right, the step along a menu bar.
func sideways(i control.Intent) int {
	if i == control.Left {
		return -1
	}
	return 1
}

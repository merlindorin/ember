package ui

import (
	"slices"
	"strings"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/internal/slot"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/control"
)

// Menu shortcuts: a menu item's aria-keyshortcuts, such as "Cmd+S" or
// "Cmd+Shift+Z" or "Backspace", chooses it when pressed, as a click would.
// Cmd is Command on macOS and Control elsewhere; Meta, Control, Shift and
// Alt name a key each. A shortcut matches whenever its Document is drawn
// and not under another Document's modal, even with nothing focused, as
// an app's menus do: the one exception to the keys being the game's. A
// focused text field keeps the keys it edits with, also one of another
// layer holding the keyboard (KeyboardCaptured), an open menu the keys
// it moves by, a modal dialog the shortcuts outside it; a matched key is
// captured. Of two Documents in one window matching a press, the one
// drawn last chooses, the frame after.

type mods uint8

const (
	modControl mods = 1 << iota
	modSuper
	modShift
	modAlt
)

// shortcut is one key and the modifiers held with it.
type shortcut struct {
	key  input.Key
	mods mods
}

// parsedKeys is an item's aria-keyshortcuts as last read: raw, for the
// platform it was read on.
type parsedKeys struct {
	raw  string
	mac  bool
	keys []shortcut
}

var keyByName = func() map[string]input.Key { //nolint:gochecknoglobals // a read-only table, built once
	m := map[string]input.Key{
		"ArrowLeft": input.KeyLeft, "ArrowRight": input.KeyRight, "ArrowUp": input.KeyUp, "ArrowDown": input.KeyDown,
		"Esc": input.KeyEscape, "Del": input.KeyDelete, "Return": input.KeyEnter, "Plus": input.KeyEqual}
	for k := input.KeySpace; k <= input.KeyF25; k++ {
		if s := k.String(); !strings.HasPrefix(s, "Key(") {
			m[strings.ToLower(s)] = k
		}
	}
	for name, k := range m {
		m[strings.ToLower(name)] = k
	}
	return m
}()

// parseShortcuts reads aria-keyshortcuts: shortcuts apart by spaces, keys
// joined by +; one naming a key it doesn't know is left out.
func parseShortcuts(s string, mac bool) []shortcut {
	var out []shortcut
	for _, one := range strings.Fields(s) {
		var sc shortcut
		ok := false
		for _, part := range strings.Split(one, "+") {
			switch strings.ToLower(part) {
			case "cmd", "command":
				if mac {
					sc.mods |= modSuper
				} else {
					sc.mods |= modControl
				}
			case "meta", "super":
				sc.mods |= modSuper
			case "control", "ctrl":
				sc.mods |= modControl
			case "shift":
				sc.mods |= modShift
			case "alt", "option":
				sc.mods |= modAlt
			default:
				sc.key, ok = keyByName[strings.ToLower(part)]
			}
		}
		if ok {
			out = append(out, sc)
		}
	}
	return out
}

// keysOf is n's shortcuts, parsed again only when its attribute or the
// platform changed.
func (d *Document) keysOf(n *Node) []shortcut {
	raw, _ := n.Attr("aria-keyshortcuts")
	mac := d.platform == textedit.Mac
	p, ok := d.parsed[n]
	if !ok || p.raw != raw || p.mac != mac {
		if d.parsed == nil {
			d.parsed = map[*Node]parsedKeys{}
		}
		p = parsedKeys{raw, mac, parseShortcuts(raw, mac)}
		d.parsed[n] = p
	}
	return p.keys
}

func held(f inputfeed.Feed) mods {
	var m mods
	for _, k := range []struct {
		key input.Key
		mod mods
	}{{input.KeyControl, modControl}, {input.KeySuper, modSuper}, {input.KeyShift, modShift}, {input.KeyAlt, modAlt}} {
		if f.RawDown(int(k.key)) {
			m |= k.mod
		}
	}
	return m
}

// shortcuts claims the item whose shortcut was pressed this frame, if any.
// The keys that would choose an item with the modifiers held now are
// captured ahead, so a game system reading them before the Document draws
// already stands aside when the key goes down.
func (d *Document) shortcuts(c *twod.Canvas, in *input.State) {
	if len(d.shortcutItems) == 0 {
		return
	}
	f := inputfeed.Of(in)
	if !f.Changed() {
		if len(d.armed) > 0 {
			f.CaptureKeys(d.armed...)
		}
		return
	}
	now := held(f)
	editing := textOf(d.machine(d.focused())) != nil || in.KeyboardCaptured()
	menuOpen := len(d.openMenus()) > 0
	d.armed = d.armed[:0]
	var chosen *Node
	for _, n := range d.shortcutItems {
		if menuDisabled(n) || len(d.modals) > 0 && d.scopeOf(n) != len(d.modals) {
			continue
		}
		for _, s := range d.keysOf(n) {
			if s.mods != now || editing && d.edits(s) || menuOpen && s.mods&^modShift == 0 {
				continue
			}
			d.armed = append(d.armed, int(s.key))
			if chosen == nil && f.RawPressed(int(s.key)) {
				chosen = n
			}
		}
	}
	if len(d.armed) > 0 {
		f.CaptureKeys(d.armed...)
	}
	if chosen != nil {
		d.claim(c, chosen)
	}
}

// edits reports whether a focused text field takes s itself: a plain or
// shifted key, or the clipboard, undo and select-all shortcuts.
func (d *Document) edits(s shortcut) bool {
	if s.mods&^modShift == 0 {
		return true
	}
	cmd := modControl
	if d.platform == textedit.Mac {
		cmd = modSuper
	}
	if s.mods == cmd {
		return slices.Contains([]input.Key{input.KeyA, input.KeyC, input.KeyV, input.KeyX, input.KeyZ, input.KeyY}, s.key)
	}
	return s.mods == cmd|modShift && s.key == input.KeyZ
}

// claims are, by window, the item a shortcut chose in the last frame, so
// the Document drawn last that frame chooses.
type claims struct{ byWin map[*platform.Window]claim }

type claim struct {
	d     *Document
	n     *Node
	frame uint64
}

func (d *Document) claim(c *twod.Canvas, n *Node) {
	w, win := c.World(), c.Window()
	if w == nil || win == nil {
		d.choose(n)
		return
	}
	cl := slot.Get[claims](w)
	if cl == nil {
		cl = &claims{byWin: map[*platform.Window]claim{}}
		slot.Set(w, cl)
	}
	cl.byWin[win] = claim{d, n, w.Time().Frame}
}

// claimed chooses the item d claimed in an earlier frame, if it was the
// last to claim one in its window.
func (d *Document) claimed(c *twod.Canvas) {
	w, win := c.World(), c.Window()
	if w == nil || win == nil {
		return
	}
	cl := slot.Get[claims](w)
	if cl == nil {
		return
	}
	k, ok := cl.byWin[win]
	now := w.Time().Frame
	switch {
	case !ok || k.frame >= now:
	case k.d == d:
		delete(cl.byWin, win)
		if !menuDisabled(k.n) {
			d.choose(k.n)
		}
	case k.frame+1 < now:
		delete(cl.byWin, win)
	}
}

// choose chooses the menu item n: a checkbox item turns, a radio item
// checks, then Clicked and every menu closes.
func (d *Document) choose(n *Node) {
	switch roleOf(n) {
	case roleMenuItemCheckbox:
		d.SetChecked(n, !d.Checked(n))
	case roleMenuItemRadio:
		d.SetChecked(n, true)
	}
	d.fire(n, control.Clicked)
	d.closeAll(n, d.focused() != nil && menuBoxOf(d.focused()) != nil)
}

// menuGroup is the radio items n checks among: those of its role=group,
// else of its menu.
func menuGroup(n *Node) []*Node {
	p := n.Parent()
	for q := p; q != nil; q = q.Parent() {
		if roleOf(q) == "group" || isMenuBox(q) {
			p = q
			break
		}
	}
	var out []*Node
	var walk func(m *Node)
	walk = func(m *Node) {
		for _, c := range m.Children() {
			switch {
			case roleOf(c) == roleMenuItemRadio:
				out = append(out, c)
			case isMenuBox(c) || roleOf(c) == "group":
			default:
				walk(c)
			}
		}
	}
	walk(p)
	return out
}

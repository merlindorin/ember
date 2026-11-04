package ui

import (
	"slices"

	"github.com/rivo/uniseg"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/emojinames"
	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// A text field with a data-shortcodes attribute, an ember extension as
// The old ember/ui's Shortcodes, turns :name: into its emoji as the closing colon
// is typed (:fire:, :+1:, :wave::skin-tone-3:). While a name of two
// characters or more is typed after a colon, it offers up to six emoji
// whose names start with it, and that the field's face draws, as options
// of the role=listbox its aria-controls names, as WAI-ARIA's combobox:
// the list shows (aria-expanded on the field), the focus stays in the
// field, Up and Down or the pad walk the options, Tab, Enter, the pad's
// south button or a click choose one, and Escape or east close the list
// until the name changes. While the list is open, Tab and the arrows
// don't move the focus.

// maxSuggestions is how many emoji the list offers.
const maxSuggestions = 6

// suggestions is a shortcode field's offer: where the colon is, the
// matches, the one picked, and the name Escape closed the list on.
type suggestions struct {
	start   int
	matches []emojinames.Entry
	picked  int
	closed  string
}

func isShortcodeField(n *Node) bool { return n != nil && !n.IsText() && has(n, "data-shortcodes") }

// suggestionList is the listbox the shortcode field n offers its matches
// in, nil for none.
func (d *Document) suggestionList(n *Node) *Node {
	id, _ := n.Attr("aria-controls")
	if l := d.byID(id); l != nil && isListbox(l) {
		return l
	}
	return nil
}

// suggestingFor is the shortcode field whose list n is in, nil for none.
func (d *Document) suggestingFor(n *Node) *Node {
	list := listboxOf(n)
	for f := range d.suggests {
		if list != nil && d.suggestionList(f) == list {
			return f
		}
	}
	return nil
}

func (d *Document) suggestOf(n *Node) *suggestions {
	if d.suggests == nil {
		d.suggests = map[*Node]*suggestions{}
	}
	sg := d.suggests[n]
	if sg == nil {
		sg = &suggestions{}
		d.suggests[n] = sg
	}
	return sg
}

// expandShortcode turns the :name: the caret closes in n into its emoji,
// or a :skin-tone-N: into the toned emoji before it.
func (d *Document) expandShortcode(n *Node, t *control.Text) {
	runes, end := t.Editor.Runes(), t.Editor.Caret()
	k := end - 1
	for k > 0 && emojinames.NameRune(runes[k-1]) {
		k--
	}
	if k < 1 || end < 1 || runes[end-1] != ':' || runes[k-1] != ':' || k == end-1 {
		return
	}
	start, name := k-1, string(runes[k:end-1])
	if emoji, ok := emojinames.Lookup(name); ok {
		t.Editor.Replace(start, end, emoji)
		d.fire(n, control.Input)
		return
	}
	if start == 0 {
		return
	}
	before := clusterStart(runes, start)
	if toned, ok := emojinames.Tone(string(runes[before:start]), name); ok {
		t.Editor.Replace(before, end, toned)
		d.fire(n, control.Input)
	}
}

// clusterStart is where the grapheme cluster ending at end starts.
func clusterStart(runes []rune, end int) int {
	text := string(runes[:end])
	at, rest, state := 0, text, -1
	last := 0
	var g string
	for len(rest) > 0 {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		last = at
		at += len([]rune(g))
	}
	return last
}

// suggest finds what the focused shortcode field n offers now, and shows
// it in its list.
func (d *Document) suggest(n *Node, t *control.Text) {
	sg := d.suggestOf(n)
	runes, caret := t.Editor.Runes(), t.Editor.Caret()
	k := caret
	for k > 0 && emojinames.NameRune(runes[k-1]) {
		k--
	}
	starts := k > 0 && runes[k-1] == ':' && (k == 1 || !emojinames.NameRune(runes[k-2]) && runes[k-2] != ':')
	switch name := string(runes[k:caret]); {
	case !starts || caret-k < 2:
		sg.matches, sg.closed = nil, ""
	case name == sg.closed:
		sg.matches = nil
	default:
		sg.closed = ""
		font := dom.Data(n).Style.Font
		draws := func(e emojinames.Entry) bool { return d.fonts.Draws(font, e.Emoji) }
		matches := emojinames.Matching(name, maxSuggestions, draws)
		if sg.start != k-1 || len(sg.matches) == 0 {
			sg.picked = 0
		}
		sg.start, sg.matches = k-1, matches
		sg.picked = min(sg.picked, max(len(matches)-1, 0))
	}
	d.showSuggestions(n, sg)
}

// closeSuggestions closes the list of the shortcode field n.
func (d *Document) closeSuggestions(n *Node) {
	if sg := d.suggests[n]; sg != nil && len(sg.matches) > 0 {
		sg.matches = nil
		d.showSuggestions(n, sg)
	}
}

// showSuggestions puts sg's matches in n's list as options, the picked
// one selected, and shows the list while there are any.
func (d *Document) showSuggestions(n *Node, sg *suggestions) {
	list := d.suggestionList(n)
	if list == nil {
		return
	}
	opts := list.Children()
	same := len(opts) == len(sg.matches)
	for i := 0; same && i < len(opts); i++ {
		same = opts[i].Text() == sg.matches[i].Emoji+" :"+sg.matches[i].Name+":"
	}
	if !same {
		for _, o := range slices.Clone(opts) {
			list.Remove(o)
		}
		for _, m := range sg.matches {
			list.Append(Div(Class("ember-suggestion"), Attr("role", roleOption),
				Span(Class("ember-suggestion-emoji"), Text(m.Emoji)), Text(" "),
				Span(Class("ember-suggestion-name"), Text(":"+m.Name+":"))))
		}
	}
	for i, o := range list.Children() {
		o.SetAttr("aria-selected", map[bool]string{true: attrTrue, false: "false"}[i == sg.picked])
	}
	open := len(sg.matches) > 0
	n.SetAttr("aria-expanded", map[bool]string{true: attrTrue, false: "false"}[open])
	if _, hidden := list.Attr("hidden"); open == hidden {
		if open {
			list.RemoveAttr("hidden")
		} else {
			list.SetAttr("hidden", "")
		}
	}
}

// placeSuggestions turns each open suggestion list above its field when
// it would cross the window's foot below and fits above (data-side
// above, which kit's sheet reads), and back below once there is room, at
// the same gap; it reports whether one turned, which needs a layout.
func (d *Document) placeSuggestions() bool {
	turned := false
	for n := range d.suggests {
		list := d.suggestionList(n)
		if list == nil || list.Parent() == nil {
			continue
		}
		if _, hidden := list.Attr("hidden"); hidden {
			continue
		}
		lb, ab := d.tree.Of[list], d.tree.Of[list.Parent()]
		if lb == nil || ab == nil {
			continue
		}
		r, a := lb.Rect(), ab.Rect()
		side, _ := list.Attr("data-side")
		gap := r.Y - (a.Y + a.H)
		if side == sideAbove {
			gap = a.Y - (r.Y + r.H)
		}
		below := a.Y+a.H+gap+r.H <= d.laidAt.Y+0.5
		above := a.Y-gap-r.H >= -0.5
		switch want := !below && above; {
		case want && side != sideAbove:
			list.SetAttr("data-side", sideAbove)
			turned = true
		case !want && side == sideAbove:
			list.RemoveAttr("data-side")
			turned = true
		}
	}
	return turned
}

// chooseSuggestion puts match i of n's list in place of the name typed.
func (d *Document) chooseSuggestion(n *Node, t *control.Text, i int) {
	sg := d.suggests[n]
	if sg == nil || i < 0 || i >= len(sg.matches) {
		return
	}
	t.Editor.Replace(sg.start, t.Editor.Caret(), sg.matches[i].Emoji)
	d.fire(n, control.Input)
	sg.matches, sg.picked = nil, 0
	d.showSuggestions(n, sg)
}

// suggestionKeys walks, chooses or closes n's open list by the keys and
// the pad, and reports whether it took them.
func (d *Document) suggestionKeys(in *input.State, n *Node, t *control.Text) bool {
	sg := d.suggests[n]
	if sg == nil || len(sg.matches) == 0 {
		return false
	}
	f := inputfeed.Of(in)
	key := func(k input.Key) bool { return f.RawPressed(int(k)) || f.RawRepeated(int(k)) }
	pad := func(b input.GamepadButton) bool { return f.RawGamepadPressed(int(b)) }
	count := len(sg.matches)
	switch {
	case key(input.KeyDown) || pad(input.GamepadDown):
		sg.picked = (sg.picked + 1) % count
	case key(input.KeyUp) || pad(input.GamepadUp):
		sg.picked = (sg.picked + count - 1) % count
	case key(input.KeyTab) || key(input.KeyEnter) || pad(input.GamepadSouth):
		d.chooseSuggestion(n, t, sg.picked)
		return true
	case key(input.KeyEscape) || pad(input.GamepadEast):
		sg.closed = string(t.Editor.Runes()[sg.start+1 : t.Editor.Caret()])
		sg.matches = nil
	default:
		return false
	}
	d.showSuggestions(n, sg)
	return true
}

// suggestionClick chooses the option n of a shortcode field's list,
// reporting whether n was one.
func (d *Document) suggestionClick(n *Node) bool {
	f := d.suggestingFor(n)
	if f == nil || !isOption(n) {
		return false
	}
	if t := textOf(d.machine(f)); t != nil {
		d.chooseSuggestion(f, t, slices.Index(listOptions(listboxOf(n)), n))
	}
	return true
}

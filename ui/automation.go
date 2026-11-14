package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/slot"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/layout"
)

// actSelectAll is the old ember/ui's act that selects a text field's text.
const actSelectAll = "select-all"

// recorded is what automation sees of an element: nothing when role is "".
type recorded struct {
	role, label string
	value       any
	enabled     bool
	acts        []string
}

// leaf is n's own part of its automation ID: its id, else its data-name,
// its aria-label, a grid row's first cell, the slug of its text (not a
// textarea's, which is its value), its label's or its placeholder; "" for
// an element automation doesn't see.
func (d *Document) leaf(n *Node) string {
	if n.ID() != "" {
		return n.ID()
	}
	if d.recorded(n).role == "" {
		return ""
	}
	if v, ok := n.Attr("data-name"); ok && v != "" {
		return v
	}
	if l, ok := n.Attr("aria-label"); ok && l != "" {
		return l
	}
	if alt, ok := n.Attr("alt"); ok && n.Tag() == dom.Img && alt != "" {
		return alt
	}
	if isGridRow(n) {
		return firstCellText(n, written)
	}
	if n.Tag() != dom.Textarea && n.Tag() != dom.Select {
		if t := visibleText(n, written); t != "" {
			return t
		}
	}
	if l := d.labelOf(n, written); l != "" {
		return l
	}
	p, _ := n.Attr("placeholder")
	return p
}

// visibleText is the text of n's subtree as an accessible name reads
// it: without what display: none or aria-hidden hides, nor tooltips (role tooltip),
// which describe an element rather than name it, nor the menus an item
// opens (role menu), whose items name themselves.
func visibleText(n *Node, text func(*Node) string) string {
	if n.IsText() {
		return text(n)
	}
	var b strings.Builder
	for _, c := range n.Children() {
		if !c.IsText() {
			if r, _ := c.Attr("role"); r == "tooltip" || r == roleMenu {
				continue
			}
			if h, _ := c.Attr("aria-hidden"); h == attrTrue {
				continue
			}
			if st := dom.Data(c).Style; st != nil && st.Display == style.DisplayNone {
				continue
			}
		}
		b.WriteString(visibleText(c, text))
	}
	return b.String()
}

// scope is the ids of n's ancestors, outermost first: the path its ID is
// under.
func scope(n *Node) []string {
	var s []string
	for p := n.Parent(); p != nil; p = p.Parent() {
		if name := scopeName(p); name != "" {
			s = append([]string{automation.Slug(name)}, s...)
		}
	}
	return s
}

// scopeName is the part of its descendants' automation IDs n stands for: its id, else its
// data-name, an automation name that, unlike an id, need not be unique in the page.
func scopeName(n *Node) string {
	if n.ID() != "" {
		return n.ID()
	}
	v, _ := n.Attr("data-name")
	return v
}

// within runs fn with c's ID scope under n's ancestors' ids.
func within(c *twod.Canvas, n *Node, fn func()) {
	prev := c.SetScope(nil)
	c.SetScope(append(append([]string(nil), prev...), scope(n)...))
	fn()
	c.RestoreScope(prev)
}

// pressOnly, pressOrSelect and pressOrSet are the acts most elements take,
// a fresh slice each time.
func pressOnly() []string     { return []string{automation.ActPress} }
func pressOrSelect() []string { return []string{automation.ActPress, automation.ActSelect} }
func pressOrSet() []string    { return []string{automation.ActPress, automation.ActSet} }

//nolint:gocyclo,cyclop,funlen // one branch per kind of element automation records
func (d *Document) recorded(n *Node) recorded {
	if n.IsText() {
		return recorded{}
	}
	enabled := !has(n, "disabled")
	if isMenuItem(n) {
		var value any
		if r := roleOf(n); r == roleMenuItemCheckbox || r == roleMenuItemRadio {
			value = d.Checked(n)
		}
		return recorded{automation.RoleMenuItem, menuLabel(n, d.shown), value, !menuDisabled(n), pressOnly()}
	}
	if isGridRow(n) {
		label, on := firstCellText(n, d.shown), enabled && !ariaDisabled(n)
		return recorded{automation.RoleSelect, label, ariaSelected(n), on, pressOrSelect()}
	}
	if isOption(n) {
		label, on := visibleText(n, d.shown), enabled && !ariaDisabled(n)
		return recorded{automation.RoleSelect, label, ariaSelected(n), on, pressOrSelect()}
	}
	if isSortHeader(n) {
		return recorded{automation.RoleButton, visibleText(n, d.shown), nil, enabled, []string{automation.ActPress}}
	}
	if isTab(n) {
		return recorded{automation.RoleSelect, visibleText(n, d.shown), tabSelected(n), enabled, pressOnly()}
	}
	if isAriaRadio(n) {
		return recorded{automation.RoleSelect, say(n, d.shown), d.radioChecked(n), enabled, []string{automation.ActPress}}
	}
	if n.Tag() == dom.Summary {
		return recorded{automation.RoleButton, visibleText(n, d.shown), nil, true, []string{automation.ActPress}}
	}
	switch m := d.machine(n).(type) {
	case *control.Button:
		return recorded{automation.RoleButton, ariaLabelOr(n, visibleText(n, d.shown)), nil, enabled, pressOnly()}
	case *control.Checkbox:
		return recorded{automation.RoleToggle, d.labelOf(n, d.shown), m.Checked, enabled, pressOrSet()}
	case *control.Select:
		label, _ := n.Attr("placeholder")
		label = layout.TranslatedAt(n, label, d.tr())
		if sh := dom.Data(n).Shadow; sh != nil {
			label = d.shown(sh)
		}
		return recorded{automation.RoleCombo, label, m.Selected, enabled, []string{automation.ActPress, automation.ActSelect}}
	case *control.Range:
		return recorded{automation.RoleSlider, d.labelOf(n, d.shown), m.Value, enabled, []string{automation.ActSet}}
	case *control.Radio:
		return recorded{automation.RoleSelect, d.labelOf(n, d.shown), m.Checked, enabled, []string{automation.ActPress}}
	case *control.Details:
		return recorded{}
	case *control.Dialog:
		if !m.Open {
			return recorded{}
		}
		return recorded{automation.RoleDialog, dialogLabel(n, d.shown), m.Modal, true, nil}
	}
	if t := textOf(d.machine(n)); t != nil {
		p := d.labelOf(n, d.shown)
		if p == "" {
			p, _ = n.Attr("placeholder")
			p = layout.TranslatedAt(n, p, d.tr())
		}
		acts := []string{automation.ActType, automation.ActSet, actSelectAll}
		if !t.Multiline() {
			acts = append(acts, automation.ActSubmit)
		}
		return recorded{automation.RoleTextInput, p, t.Shown(), enabled && !t.Editor.ReadOnly, acts}
	}
	if r, ok := d.ariaWidget(n); ok {
		return r
	}
	switch n.Tag() {
	case dom.Progress:
		var value any
		if v, ok := n.Attr("value"); ok {
			value = progressShare(v, number(n, "max", 1))
		}
		return recorded{automation.RoleBar, ariaLabelOr(n, d.labelOf(n, d.shown)), value, true, nil}
	case dom.Meter:
		return recorded{automation.RoleMeter, ariaLabelOr(n, d.labelOf(n, d.shown)), number(n, "value", 0), true, nil}
	case dom.Img:
		alt, _ := n.Attr("alt")
		if alt != "" || n.ID() != "" {
			return recorded{automation.RoleLabel, ariaLabelOr(n, layout.TranslatedAt(n, alt, d.tr())), nil, true, nil}
		}
	case dom.TextTag, dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
		dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label,
		dom.Button, dom.Input, dom.Select, dom.Option, dom.Textarea, dom.Details, dom.Summary, dom.Dialog, dom.Table,
		dom.Thead, dom.Tbody, dom.Tr, dom.Th, dom.Td, dom.Hr, dom.Canvas, dom.Kbd, dom.Samp, dom.Pre, dom.Caption,
		dom.Tfoot:
		// the others record by their role, tabindex or text below
	}
	if has(n, "tabindex") {
		return recorded{automation.RoleButton, visibleText(n, d.shown), nil, enabled, []string{automation.ActPress}}
	}
	if n.ID() != "" && n != d.html && n != d.body || textOnly(n) && scopeName(n) != "" {
		return recorded{automation.RoleLabel, visibleText(n, d.shown), nil, true, nil}
	}
	return recorded{}
}

// textOnly reports whether n holds text and no element: a label a
// data-name names, recorded as the old ember/ui records a named Label.
func textOnly(n *Node) bool {
	for _, c := range n.Children() {
		if !c.IsText() {
			return false
		}
	}
	return len(n.Children()) > 0
}

// labelOf is the text of the label for n, if any: one round it, as
// HTML's implicit label, or one whose for names it; n's own text, a
// select's options, left out.
func (d *Document) labelOf(n *Node, text func(*Node) string) string {
	text = without(n, text)
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Tag() == dom.Label {
			if _, ok := p.Attr("for"); !ok && d.labelled(p) == n {
				return say(p, text)
			}
		}
	}
	if n.ID() == "" {
		return ""
	}
	label := ""
	d.html.Walk(func(l *Node) bool {
		if f, ok := l.Attr("for"); ok && l.Tag() == dom.Label && f == n.ID() {
			label = say(l, text)
		}
		return label == ""
	})
	return label
}

// without is text, but "" for the text nodes under n.
func without(n *Node, text func(*Node) string) func(*Node) string {
	return func(c *Node) string {
		for p := c.Parent(); p != nil; p = p.Parent() {
			if p == n {
				return ""
			}
		}
		return text(c)
	}
}

// record publishes every element automation sees: not those without a
// box, display: none and what is under them, nor display: contents
// itself, whose children are seen.
func (d *Document) record(c *twod.Canvas) {
	r := slot.Get[automation.Registry](c.World())
	if !r.Recording() {
		return
	}
	d.html.Walk(func(n *Node) bool {
		if n.IsText() {
			return false
		}
		if st := dom.Data(n).Style; st != nil && st.Display == style.Contents {
			return true
		}
		if n.Box() == (geom.Rect{}) && n != d.html {
			return false
		}
		rec := d.recorded(n)
		if leaf := d.leaf(n); rec.role != "" && leaf != "" {
			within(c, n, func() {
				if n == d.focused() {
					c.SetFocused(true)
				}
				c.Record(leaf, rec.role, rec.label, rec.value, rec.enabled, d.toWindow(n.Box()), pointerActs(n, rec.acts)...)
				c.SetFocused(false)
			})
		}
		return true
	})
}

// acts does what automation clients asked of the elements this frame.
func (d *Document) acts(c *twod.Canvas) {
	r := slot.Get[automation.Registry](c.World())
	if !r.Recording() {
		return
	}
	d.html.Walk(func(n *Node) bool {
		if n.IsText() {
			return false
		}
		leaf := d.leaf(n)
		if leaf == "" {
			return true
		}
		var act *automation.Act
		within(c, n, func() { act = r.Take(c.ID(leaf)) })
		if act != nil {
			d.act(n, act, func(id string) *Node { return d.byPath(c, id) })
		}
		return true
	})
}

// pointerActs are acts with drag for a Draggable element and context for
// one that it or an ancestor takes OnContextMenu for (decision 0062).
func pointerActs(n *Node, acts []string) []string {
	if draggableIn([]*Node{n}) != nil {
		acts = append(slices.Clip(acts), automation.ActDrag)
	}
	if slices.ContainsFunc(chain(n), func(a *Node) bool { return handles(a, EventContext) }) {
		acts = append(slices.Clip(acts), automation.ActContext)
	}
	return acts
}

// byPath is the element whose automation path is id, nil for none.
func (d *Document) byPath(c *twod.Canvas, id string) *Node {
	var found *Node
	d.html.Walk(func(n *Node) bool {
		if n.IsText() || found != nil {
			return false
		}
		if leaf := d.leaf(n); leaf != "" {
			within(c, n, func() {
				if c.ID(leaf) == id {
					found = n
				}
			})
		}
		return found == nil
	})
	return found
}

func (d *Document) act(n *Node, a *automation.Act, byPath func(string) *Node) {
	if (isGridRow(n) || isOption(n)) && (a.Kind == automation.ActPress || a.Kind == automation.ActSelect) {
		d.actOnItem(n, a.Kind == automation.ActPress)
		return
	}
	m := d.machine(n)
	t := textOf(m)
	switch a.Kind {
	case automation.ActPress:
		d.moveFocus(func() bool { return d.focus.Set(n, false) })
		d.click(n)
	case automation.ActSelect:
		if s, ok := m.(*control.Select); ok {
			if i, found := optionIndex(s, a.Value); found {
				if !s.Open {
					s.Click()
				}
				d.fire(n, s.ClickOption(i))
				s.ClickOutside()
			}
		}
	case automation.ActSet:
		switch m := m.(type) {
		case *control.Checkbox:
			if on, ok := a.Value.(bool); ok && on != m.Checked {
				d.fire(n, m.Click())
			}
		case *control.Range:
			if v, ok := toFloat(a.Value); ok {
				d.fire(n, m.SetValue(v)|control.Changed)
			}
		}
		if s, ok := a.Value.(string); ok && t != nil {
			t.Focus()
			t.Editor.Do(textedit.SelectAll, false, nil, nil)
			e := t.Type(s)
			if s == "" {
				e |= t.Key(textedit.Backspace, false, nil)
			}
			d.fire(n, e|t.Blur())
		}
	case automation.ActContext:
		d.events = append(d.events, ContextClicked{Node: n})
	case automation.ActDrag:
		id, _ := a.Value.(string)
		if target := byPath(id); target != nil && handles(target, EventDrop) {
			d.events = append(d.events, Dropped{Target: target, Dragged: n}, DragEnded{Source: n, Dropped: true})
		}
	case automation.ActType:
		if s, ok := a.Value.(string); ok && t != nil {
			d.fire(n, t.Type(s))
		}
	case actSelectAll:
		if t != nil {
			t.Editor.Do(textedit.SelectAll, false, nil, nil)
		}
	case automation.ActSubmit:
		if t != nil {
			d.fire(n, t.Key(textedit.Submit, false, nil))
		}
	}
}

func optionIndex(s *control.Select, v any) (int, bool) {
	if f, ok := toFloat(v); ok && int(f) >= -1 && int(f) < len(s.Options) {
		return int(f), true
	}
	if str, ok := v.(string); ok {
		for i, o := range s.Options {
			if o.Label == str || o.Value == str || automation.Slug(o.Label) == automation.Slug(str) {
				return i, true
			}
		}
	}
	return 0, false
}

func toFloat(v any) (float64, bool) {
	switch v := v.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}

// dialogLabel names a dialog: its aria-label, else its first heading's text.
func dialogLabel(n *Node, text func(*Node) string) string {
	if l, ok := n.Attr("aria-label"); ok {
		return l
	}
	label := ""
	n.Walk(func(c *Node) bool {
		if t := c.Tag(); label == "" && (t == dom.H1 || t == dom.H2 || t == dom.H3 || t == dom.H4) {
			label = say(c, text)
		}
		return label == ""
	})
	return label
}

// ariaLabelOr is n's aria-label, or text without one: an icon button's name.
func ariaLabelOr(n *Node, text string) string {
	if v, ok := n.Attr("aria-label"); ok && v != "" {
		return v
	}
	return text
}

// actOnItem is automation's select or press on a grid row or listbox option, as the old ember/ui's:
// select selects it, emitting Changed for its grid or listbox; press selects it and
// activates it, emitting Submitted too. A dimmed one takes neither.
func (d *Document) actOnItem(n *Node, press bool) {
	if ariaDisabled(n) {
		return
	}
	var box *Node
	var changed bool
	if isGridRow(n) {
		box = gridOf(n)
		changed = selectRow(box, n)
	} else {
		box = listboxOf(n.Parent())
		changed = selectOption(box, n)
	}
	d.moveFocus(func() bool { return d.focus.Set(box, false) })
	if changed {
		d.fire(box, control.Changed)
	}
	if press {
		d.fire(box, control.Submitted)
	}
}

// ariaWidget records an element whose ARIA role is a meter, a progress bar
// or a slider by its aria-valuenow, as Chrome's accessibility tree does: a
// game's canvas says what it shows. A progress bar reads its share of
// aria-valuemin to aria-valuemax (0 to 100 by default), as a <progress>
// reads value over max; a meter and a slider read the value as is.
func (d *Document) ariaWidget(n *Node) (recorded, bool) {
	role := roleOf(n)
	if role != "meter" && role != roleProgressbar && role != "slider" {
		return recorded{}, false
	}
	label := ariaLabelOr(n, d.labelOf(n, d.shown))
	var value any
	if _, ok := n.Attr("aria-valuenow"); ok {
		now := number(n, "aria-valuenow", 0)
		value = now
		if role == roleProgressbar {
			lo, hi := number(n, "aria-valuemin", 0), number(n, "aria-valuemax", 100)
			if hi > lo {
				value = min(max((now-lo)/(hi-lo), 0), 1)
			}
		}
	}
	switch role {
	case "meter":
		return recorded{automation.RoleMeter, label, value, true, nil}, true
	case roleProgressbar:
		return recorded{automation.RoleBar, label, value, true, nil}, true
	}
	return recorded{automation.RoleSlider, label, value, !ariaDisabled(n), nil}, true
}

// progressShare is a progress's value over its max, 0 to 1, as HTML reads
// them.
func progressShare(value string, maxV float64) float64 {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil || maxV <= 0 {
		return 0
	}
	return min(max(v/maxV, 0), 1)
}

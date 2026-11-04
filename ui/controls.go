package ui

import (
	"math"
	"strconv"

	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// machine is n's control, made from its attributes the first time it is
// needed; nil for an element without behaviour.
func (d *Document) machine(n *Node) any {
	if n == nil || n.IsText() {
		return nil
	}
	if m, ok := d.machines[n]; ok {
		return m
	}
	m := newMachine(n)
	if m != nil {
		if d.machines == nil {
			d.machines = map[*Node]any{}
		}
		d.machines[n] = m
	}
	return m
}

func has(n *Node, name string) bool { _, ok := n.Attr(name); return ok }

func number(n *Node, name string, or float64) float64 {
	if s, ok := n.Attr(name); ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return or
}

//nolint:funlen // one case per tag that has a control
func newMachine(n *Node) any {
	value, _ := n.Attr("value")
	disabled := has(n, "disabled")
	switch n.Tag() {
	case dom.Canvas:
		if m, ok := dom.Data(n).Canvas.(*control.Canvas); ok {
			return m
		}
		return &control.Canvas{}
	case dom.Button:
		return &control.Button{Disabled: disabled}
	case dom.Textarea:
		t := control.NewTextArea(n.Text())
		t.SetReadOnly(has(n, "readonly"))
		return t
	case dom.Details:
		return &control.Details{Open: has(n, "open")}
	case dom.Dialog:
		return &control.Dialog{Open: has(n, "open")}
	case dom.Select:
		var opts []control.Option
		selected := -1
		for _, c := range n.Children() {
			if c.Tag() != dom.Option {
				continue
			}
			v, _ := c.Attr("value")
			opts = append(opts, control.Option{Label: c.Text(), Value: v, Disabled: has(c, "disabled")})
			if has(c, "selected") {
				selected = len(opts) - 1
			}
		}
		s := control.NewSelect(opts...)
		if selected >= 0 || has(n, "placeholder") {
			s.Selected, s.Highlight = selected, selected
		}
		s.Disabled = disabled
		return s
	case dom.Input:
		typ, _ := n.Attr("type")
		var t *control.Text
		switch typ {
		case "checkbox":
			return &control.Checkbox{Checked: has(n, "checked"), Disabled: disabled}
		case "radio":
			return &control.Radio{Checked: has(n, "checked"), Disabled: disabled}
		case "range":
			r := &control.Range{
				Min: number(n, "min", 0), Max: number(n, "max", 100), Step: number(n, "step", 1), Disabled: disabled,
			}
			r.SetValue(number(n, "value", r.Min+(r.Max-r.Min)/2))
			return r
		case "number":
			num := control.NewNumber(value)
			num.Min, num.Max, num.Step = number(n, "min", math.NaN()), number(n, "max", math.NaN()), number(n, "step", 1)
			if v, _ := n.Attr("step"); v == "any" {
				num.Step = 0
			}
			num.SetReadOnly(has(n, "readonly"))
			return num
		case "search":
			t = control.NewSearch(value)
		case "password":
			t = control.NewPassword(value)
		default:
			t = control.NewText(value)
		}
		t.SetReadOnly(has(n, "readonly"))
		return t
	case dom.TextTag, dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
		dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label,
		dom.Option, dom.Summary, dom.Table, dom.Thead, dom.Tbody, dom.Tr, dom.Th, dom.Td, dom.Progress, dom.Meter,
		dom.Img, dom.Hr, dom.Kbd, dom.Samp, dom.Pre, dom.Caption, dom.Tfoot:
		// no control of their own
	}
	return nil
}

// textOf is a text field's editing machine, a number's included.
func textOf(m any) *control.Text {
	switch m := m.(type) {
	case *control.Text:
		return m
	case *control.Number:
		return &m.Text
	}
	return nil
}

// sync shows n's machine in its state, attributes and shadow text.
func (d *Document) sync(n *Node, m any) {
	disabled := has(n, "disabled")
	n.SetState(dom.Disabled, disabled)
	switch m := m.(type) {
	case *control.Button:
		m.Disabled = disabled
	case *control.Range:
		m.Disabled = disabled
	}
	switch m := m.(type) {
	case *control.Checkbox:
		m.Disabled = disabled
		n.SetState(dom.Checked, m.Checked)
	case *control.Radio:
		m.Disabled = disabled
		radioState(n, m.Checked)
	case *control.Dialog:
		n.SetState(dom.Open, m.Open)
		n.SetState(dom.Modal, m.Open && m.Modal)
		if m.Open {
			n.SetAttr("open", "")
		} else {
			n.RemoveAttr("open")
		}
	case *control.Details:
		n.SetState(dom.Open, m.Open)
		if m.Open {
			n.SetAttr("open", "")
		} else {
			n.RemoveAttr("open")
		}
	case *control.Select:
		m.Disabled = disabled
		n.SetState(dom.Open, m.Open)
		d.showOptions(n, m)
		if m.Selected >= 0 && m.Selected < len(m.Options) {
			n.SetShadow(m.Options[m.Selected].Label, false)
		} else {
			p, _ := n.Attr("placeholder")
			n.SetShadow(p, true)
		}
	}
	if t := textOf(m); t != nil {
		t.SetReadOnly(has(n, "readonly"))
		if _, num := m.(*control.Number); !num {
			t.SetMaxLength(max(int(number(n, "maxlength", 0)), 0))
		}
		if t.Empty() {
			p, _ := n.Attr("placeholder")
			n.SetShadow(p, true)
		} else {
			shown, _ := d.shownText(n, t)
			n.SetShadow(shown, false)
		}
	}
}

// Value is a control's value now: a text field's or textarea's text, the
// selected option's value, a range's number, "on" for a checked checkbox.
func (d *Document) Value(n *Node) string {
	switch m := d.machine(n).(type) {
	case *control.Checkbox:
		if m.Checked {
			return "on"
		}
		return ""
	case *control.Radio:
		if !m.Checked {
			return ""
		}
		if v, ok := n.Attr("value"); ok {
			return v
		}
		return "on"
	case *control.Select:
		return m.Value()
	case *control.Range:
		return strconv.FormatFloat(m.Value, 'g', -1, 64)
	}
	if t := textOf(d.machine(n)); t != nil {
		return t.Value()
	}
	return ""
}

// SetValue sets a control's value, as the game does, without events. A
// focused text field keeps what the player is typing, until it changes or
// is submitted: then it takes the game's answer, as a chat clearing its
// field on Enter.
func (d *Document) SetValue(n *Node, v string) {
	switch m := d.machine(n).(type) {
	case *control.Select:
		for i, o := range m.Options {
			if o.Value == v || o.Value == "" && o.Label == v {
				m.Selected, m.Highlight = i, i
			}
		}
	case *control.Range:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			m.SetValue(f)
		}
	case *control.Checkbox:
		m.Checked = v != ""
	default:
		if t := textOf(m); t != nil && (d.focused() != n || d.typing != n) {
			t.SetValue(v)
		}
	}
	d.sync(n, d.machine(n))
}

// Checked reports whether a checkbox or a radio is checked.
func (d *Document) Checked(n *Node) bool {
	if isMenuItem(n) {
		v, _ := n.Attr("aria-checked")
		return v == attrTrue
	}
	if d.isRadio(n) {
		return d.radioChecked(n)
	}
	c, ok := d.machine(n).(*control.Checkbox)
	return ok && c.Checked
}

// SetChecked checks or unchecks a checkbox, or a radio (checking one
// unchecks the rest of its group), without events.
func (d *Document) SetChecked(n *Node, on bool) {
	if roleOf(n) == roleMenuItemRadio && on {
		for _, o := range menuGroup(n) {
			o.SetAttr("aria-checked", strconv.FormatBool(o == n))
		}
		return
	}
	if isMenuItem(n) {
		n.SetAttr("aria-checked", strconv.FormatBool(on))
		return
	}
	if d.isRadio(n) {
		if on {
			for _, o := range d.radioGroup(n) {
				d.setRadio(o, o == n)
			}
		} else {
			d.setRadio(n, false)
		}
		return
	}
	if c, ok := d.machine(n).(*control.Checkbox); ok {
		c.Checked, c.Indeterminate = on, false
		d.sync(n, c)
	}
}

// Selected is a select's chosen option, a tablist's selected tab, a
// grid's selected row or a listbox's selected option, -1 for none.
func (d *Document) Selected(n *Node) int {
	if s, ok := d.machine(n).(*control.Select); ok {
		return s.Selected
	}
	if roleOf(n) == roleTablist {
		return tabIndexOf(n)
	}
	if isGrid(n) {
		return selectedRow(n)
	}
	if isListbox(n) {
		return selectedOption(n)
	}
	return -1
}

// SetSelected chooses a select's option i, -1 for none, or selects a
// tablist's tab i, a grid's row i (-1 for none) or a listbox's option
// i, without events.
func (d *Document) SetSelected(n *Node, i int) {
	if s, ok := d.machine(n).(*control.Select); ok {
		s.Selected, s.Highlight = i, i
		d.sync(n, s)
		return
	}
	if tabs := tabsOf(n); roleOf(n) == roleTablist && i >= 0 && i < len(tabs) {
		d.selectTab(tabs[i])
	}
	if rows := gridRows(n); isGrid(n) && i < len(rows) {
		if i < 0 {
			for _, r := range rows {
				r.SetAttr("aria-selected", "false")
			}
		} else {
			selectRow(n, rows[i])
			d.revealing = rows[i]
		}
	}
	if opts := listOptions(n); isListbox(n) && i >= 0 && i < len(opts) {
		selectOption(n, opts[i])
	}
}

// showOptions gives a select's options their labels as the player reads
// them, which its list shows and typing matches.
func (d *Document) showOptions(n *Node, m *control.Select) {
	i := 0
	for _, c := range n.Children() {
		if c.Tag() != dom.Option || i >= len(m.Options) {
			continue
		}
		m.Options[i].Shown = ""
		if d.translator != nil {
			m.Options[i].Shown = say(c, d.shown)
		}
		i++
	}
}

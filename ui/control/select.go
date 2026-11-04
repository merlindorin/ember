package control

import (
	"strings"
	"unicode"
)

// Option is one choice of a select.
type Option struct {
	Label, Value string
	Disabled     bool
	// Shown is the label as the player reads it, translated; Label when empty.
	Shown string
}

// Text is what the list shows of the option.
func (o Option) Text() string {
	if o.Shown != "" {
		return o.Shown
	}
	return o.Label
}

// value is what a form reads of the option: its Value, else its label.
func (o Option) value() string {
	if o.Value != "" {
		return o.Value
	}
	return o.Label
}

// Select is a select element, shown as a pop-up list: Activate or a
// click opens it on the selected option, the arrows walk its enabled
// options, Activate or a click chooses one, and Back or a click outside
// closes it as it was. Typing jumps to the next option starting with
// that letter, open or closed.
type Select struct {
	Options  []Option
	Disabled bool
	// Selected is the chosen option, -1 for none; Highlight the one the
	// open list is on.
	Selected, Highlight int
	Open                bool
}

func NewSelect(options ...Option) *Select {
	s := &Select{Options: options, Selected: -1}
	s.Selected = s.next(-1, 1)
	s.Highlight = s.Selected
	return s
}

// Value is the selected option's value, "" for none.
func (s *Select) Value() string {
	if s.Selected < 0 || s.Selected >= len(s.Options) {
		return ""
	}
	return s.Options[s.Selected].value()
}

// Handles reports whether i is the select's: Activate always, and the
// arrows and Back only while it is open.
func (s *Select) Handles(i Intent) bool {
	if s.Disabled {
		return false
	}
	switch i {
	case Activate:
		return true
	case Up, Down, Home, End, PageUp, PageDown, Back:
		return s.Open
	case None, Left, Right:
	}
	return false
}

func (s *Select) Do(i Intent) Events {
	if !s.Handles(i) {
		return 0
	}
	if !s.Open {
		s.open()
		return 0
	}
	switch i {
	case Activate:
		return s.choose(s.Highlight)
	case Back:
		s.Open = false
	case Down:
		s.Highlight = s.next(s.Highlight, 1)
	case Up:
		s.Highlight = s.next(s.Highlight, -1)
	case Home, PageUp:
		s.Highlight = s.next(-1, 1)
	case End, PageDown:
		s.Highlight = s.next(len(s.Options), -1)
	case None, Left, Right: // Handles lets only these through
	}
	return 0
}

// Click opens or closes the list from the select itself.
func (s *Select) Click() Events {
	switch {
	case s.Disabled:
	case s.Open:
		s.Open = false
	default:
		s.open()
	}
	return 0
}

// ClickOption chooses option i of the open list, unless it is disabled.
func (s *Select) ClickOption(i int) Events {
	if !s.Open || !s.enabled(i) {
		return 0
	}
	return s.choose(i)
}

// ClickOutside closes the list as it was.
func (s *Select) ClickOutside() { s.Open = false }

// TypeAhead goes to the next enabled option after the current one whose
// label starts with text, any case, wrapping: the highlight while open,
// the choice while closed.
func (s *Select) TypeAhead(text string) Events {
	if s.Disabled || text == "" {
		return 0
	}
	from := s.Selected
	if s.Open {
		from = s.Highlight
	}
	n := len(s.Options)
	for k := 1; k <= n; k++ {
		i := (from + k + n) % n
		if s.enabled(i) && strings.HasPrefix(strings.ToLower(s.Options[i].Text()), strings.Map(unicode.ToLower, text)) {
			if s.Open {
				s.Highlight = i
				return 0
			}
			return s.pick(i)
		}
	}
	return 0
}

func (s *Select) open() {
	s.Open, s.Highlight = true, s.Selected
	if !s.enabled(s.Highlight) {
		s.Highlight = s.next(-1, 1)
	}
}

func (s *Select) choose(i int) Events {
	s.Open = false
	return s.pick(i)
}

func (s *Select) pick(i int) Events {
	if !s.enabled(i) || i == s.Selected {
		return 0
	}
	s.Selected, s.Highlight = i, i
	return Input | Changed
}

func (s *Select) enabled(i int) bool { return i >= 0 && i < len(s.Options) && !s.Options[i].Disabled }

// next is the first enabled option from i going by dir, or i if none.
func (s *Select) next(i, dir int) int {
	for j := i + dir; j >= 0 && j < len(s.Options); j += dir {
		if s.enabled(j) {
			return j
		}
	}
	if s.enabled(i) {
		return i
	}
	return -1
}

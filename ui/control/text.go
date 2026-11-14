package control

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/internal/textedit"
)

type textKind int

const (
	plainText textKind = iota
	searchText
	passwordText
	areaText
)

// Text is an input of type text, search or password, or a textarea: its
// editor, with Input on every edit and Changed when an edit is committed,
// by Enter or by losing the focus. While focused it takes the keyboard:
// ui hands it keys as textedit actions and typed text.
type Text struct {
	Editor *textedit.Editor
	// Layout is where the text lies once laid out, for the line moves of
	// a textarea; nil is one line.
	Layout textedit.Layout
	kind   textKind
	// committed is the value at the last commit, or when the focus came.
	committed string
	// shown is Shown's last string, and shownOf the text it was made for.
	shown   string
	shownOf []rune
}

func NewText(value string) *Text { return newText(textedit.New(value), plainText) }

// NewSearch is a search input: Back clears it.
func NewSearch(value string) *Text { return newText(textedit.New(value), searchText) }

// NewPassword is a password input: it shows bullets and never copies.
func NewPassword(value string) *Text {
	t := newText(textedit.New(value), passwordText)
	t.Editor.Password = true
	return t
}

// NewTextArea is a textarea: Enter breaks the line, and the submit
// shortcut (Command or Control+Enter) submits.
func NewTextArea(value string) *Text { return newText(textedit.NewArea(value), areaText) }

func newText(e *textedit.Editor, k textKind) *Text {
	return &Text{Editor: e, kind: k, committed: e.Text()}
}

func (t *Text) Value() string { return t.Editor.Text() }

// Empty reports whether the field holds no text, without making a string.
func (t *Text) Empty() bool { return len(t.Editor.Runes()) == 0 }

// SetValue sets the text as a game does, without an event: what it sets
// is the committed value.
func (t *Text) SetValue(v string) {
	t.Editor.SetText(v)
	t.Editor.Place(len(t.Editor.Runes()), false)
	t.committed = t.Value()
}

func (t *Text) SetMaxLength(n int)   { t.Editor.MaxLength = n }
func (t *Text) SetReadOnly(on bool)  { t.Editor.ReadOnly = on }
func (t *Text) Focus()               { t.committed = t.Value() }
func (t *Text) Blur() Events         { return t.commit() }
func (t *Text) Multiline() bool      { return t.kind == areaText }
func (t *Text) Type(s string) Events { return when(t.Editor.Type(s), Input) }
func (t *Text) Password() bool       { return t.kind == passwordText }
func (t *Text) Search() bool         { return t.kind == searchText }

// Shown is the text as drawn: a bullet per character for a password. It
// is the same string while the text is the same, made only on a change.
func (t *Text) Shown() string {
	r := t.Editor.Runes()
	if t.shownOf != nil && slices.Equal(r, t.shownOf) {
		return t.shown
	}
	t.shownOf = append(t.shownOf[:0:0], r...)
	if t.Password() {
		t.shown = strings.Repeat("•", len(r))
	} else {
		t.shown = string(r)
	}
	return t.shown
}

func (t *Text) commit() Events {
	changed := t.Value() != t.committed
	t.committed = t.Value()
	return when(changed, Changed)
}

// Key performs action, the selection extended when extend, copying and
// pasting through clip. Enter submits a one-line input and commits it; in
// a textarea only the submit shortcut submits, and the commit waits for
// the focus to leave.
func (t *Text) Key(a textedit.Action, extend bool, clip textedit.Clipboard) Events {
	switch {
	case a == textedit.Submit && t.Multiline():
		return Submitted
	case a == textedit.Submit, a == textedit.Newline && !t.Multiline():
		return Submitted | t.commit()
	}
	return when(t.Editor.Do(a, extend, t.Layout, clip), Input)
}

// Handles reports whether i, from a pad or arrows, edits the text rather
// than moving the focus: the caret's moves along the line, the line moves
// of a textarea, and Back on a search with text in it.
func (t *Text) Handles(i Intent) bool {
	switch i {
	case Left, Right, Home, End:
		return true
	case Up, Down, PageUp, PageDown:
		return t.Multiline()
	case Back:
		return t.Search() && t.Value() != ""
	case None, Activate:
	}
	return false
}

func (t *Text) Do(i Intent) Events {
	if !t.Handles(i) {
		return 0
	}
	if i == Back {
		t.Editor.Select(0, len(t.Editor.Runes()))
		return when(t.Editor.Type(""), Input)
	}
	return t.Key(editAction(i), false, nil)
}

// editAction is the editor's move for intent i, textedit.None for any
// intent that doesn't move the caret.
func editAction(i Intent) textedit.Action {
	switch i {
	case Left:
		return textedit.Left
	case Right:
		return textedit.Right
	case Home:
		return textedit.LineStart
	case End:
		return textedit.LineEnd
	case Up:
		return textedit.Up
	case Down:
		return textedit.Down
	case PageUp:
		return textedit.PageUp
	case PageDown:
		return textedit.PageDown
	case None, Activate, Back: // they don't move the caret
	}
	return textedit.None
}

// Number is an input of type number: text of number characters, stepped
// by Up and Down within Min and Max (NaN for none) on Step from Min.
type Number struct {
	Text
	Min, Max, Step float64
}

func NewNumber(value string) *Number {
	n := &Number{Text: *NewText(""), Min: math.NaN(), Max: math.NaN(), Step: 1}
	n.SetValue(value)
	return n
}

// Type keeps only what a number is written with.
func (n *Number) Type(s string) Events {
	return n.Text.Type(strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || strings.ContainsRune("+-.eE", r) {
			return r
		}
		return -1
	}, s))
}

// Number is the value, if it reads as one.
func (n *Number) Number() (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(n.Value()), 64)
	return v, err == nil && !math.IsInf(v, 0) && !math.IsNaN(v)
}

func (n *Number) Handles(i Intent) bool {
	return i == Up || i == Down || n.Text.Handles(i)
}

// Settle readies a commit: the number kept within Min and Max on a step,
// and what doesn't read as one back to the committed value.
func (n *Number) Settle() {
	s := n.committed
	if v, ok := n.Number(); ok {
		s = strconv.FormatFloat(snap(v, n.Min, n.Max, n.Step), 'f', -1, 64)
	}
	if s != n.Value() {
		n.Editor.SetText(s)
		n.Editor.Place(len(n.Editor.Runes()), false)
	}
}

// Do steps the number by Up and Down: to the next step that way, or by a
// step when already on one. An empty or unreadable input counts as 0.
func (n *Number) Do(i Intent) Events {
	if i != Up && i != Down {
		return n.Text.Do(i)
	}
	if n.Editor.ReadOnly {
		return 0
	}
	v, _ := n.Number()
	step := n.Step
	if step <= 0 {
		step = 1
	}
	base := n.Min
	if math.IsNaN(base) {
		base = 0
	}
	k := (v - base) / step
	switch {
	case math.Abs(k-math.Round(k)) < 1e-9:
		k = math.Round(k)
		if i == Up {
			k++
		} else {
			k--
		}
	case i == Up:
		k = math.Ceil(k)
	default:
		k = math.Floor(k)
	}
	v = snap(base+k*step, n.Min, n.Max, step)
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if s == n.Value() {
		return 0
	}
	committed := n.committed
	n.SetValue(s)
	n.committed = committed
	return Input
}

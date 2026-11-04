// Package textedit is the editing core of ui's text inputs and areas:
// text, caret and selection by grapheme cluster, words and lines, the
// keys of each platform, the clipboard, and undo. It is pure Go, and knows
// nothing of drawing: a Layout tells it where lines are.
package textedit

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// Layout is where the text lies once laid out: the lines, wrapped or not,
// that up, down and the line keys move along, and the mouse lands on.
// Positions are rune indexes.
type Layout interface {
	// Lines is how many visual lines the text has.
	Lines() int
	// Line is the visual line holding position i.
	Line(i int) int
	// Span is the positions line covers, end before a line break.
	Span(line int) (start, end int)
	// X is how far position i is along its line.
	X(i int) float32
	// At is the position on line nearest x.
	At(line int, x float32) int
	// Page is how many lines PageUp and PageDown move.
	Page() int
}

// Clipboard is where copy and cut put text and paste takes it.
type Clipboard interface {
	Clipboard() string
	SetClipboard(string)
}

type state struct {
	text          []rune
	caret, anchor int
}

// Editor is text being edited: its caret and the other end of its
// selection, anchor, both rune positions on cluster boundaries.
type Editor struct {
	text          []rune
	caret, anchor int
	goal          float32
	hasGoal       bool
	undo, redo    []state
	typing        bool

	// Multiline takes line breaks and tabs; MaxLength limits the text to
	// that many characters; ReadOnly only selects and copies; Password
	// copies and cuts nothing.
	Multiline bool
	MaxLength int
	ReadOnly  bool
	Password  bool
}

// New is an editor of one line of text, its caret at the end.
func New(text string) *Editor { return newEditor(text, false) }

// NewArea is an editor of text that takes line breaks and tabs.
func NewArea(text string) *Editor { return newEditor(text, true) }

func newEditor(text string, multiline bool) *Editor {
	e := &Editor{Multiline: multiline}
	e.SetText(text)
	e.caret = len(e.text)
	e.anchor = e.caret
	return e
}

func (e *Editor) Text() string  { return string(e.text) }
func (e *Editor) Runes() []rune { return e.text }
func (e *Editor) Caret() int    { return e.caret }

// Selection is the selected positions, from before to after; empty when
// they are equal.
func (e *Editor) Selection() (int, int) { return min(e.caret, e.anchor), max(e.caret, e.anchor) }

// Selected is the selected text.
func (e *Editor) Selected() string {
	from, to := e.Selection()
	return string(e.text[from:to])
}

// SetText replaces the text without an undo step, keeping the caret and
// selection inside it.
func (e *Editor) SetText(text string) {
	e.text = []rune(e.clean(text))
	e.caret, e.anchor = e.fit(e.caret), e.fit(e.anchor)
	e.typing = false
}

// Place puts the caret at i, extending the selection when extend.
func (e *Editor) Place(i int, extend bool) {
	e.caret = e.fit(i)
	if !extend {
		e.anchor = e.caret
	}
	e.hasGoal, e.typing = false, false
}

// Select selects from from to to, the caret at to.
func (e *Editor) Select(from, to int) {
	e.anchor, e.caret = e.fit(from), e.fit(to)
	e.hasGoal, e.typing = false, false
}

func (e *Editor) fit(i int) int { return min(max(i, 0), len(e.text)) }

// clean is text as the editor takes it: control characters go, and so do
// line breaks and tabs unless multiline, where a one-line input has a
// space instead.
func (e *Editor) clean(text string) string {
	var b strings.Builder
	for _, r := range strings.ReplaceAll(text, "\r\n", "\n") {
		switch {
		case r == '\n' || r == '\t':
			if e.Multiline {
				b.WriteRune(r)
			} else {
				b.WriteRune(' ')
			}
		case r < 32 || r == 127 || r >= 0x80 && r < 0xa0:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Type puts text in place of the selection, as typed: a run of typing is
// one undo step.
func (e *Editor) Type(text string) bool {
	if !e.typing {
		e.save()
	}
	changed := e.replace([]rune(e.clean(text)))
	e.typing = true
	return changed
}

// Replace puts text in place of positions from to to as one undo step,
// the caret after it: a shortcode turned into its emoji.
func (e *Editor) Replace(from, to int, text string) bool {
	e.save()
	e.Select(from, to)
	changed := e.replace([]rune(e.clean(text)))
	e.typing = false
	return changed
}

// replace puts r in place of the selection, as much of it as MaxLength
// leaves room for, whole clusters only.
func (e *Editor) replace(r []rune) bool {
	if e.ReadOnly {
		return false
	}
	from, to := e.Selection()
	if e.MaxLength > 0 {
		room := e.MaxLength - (len(e.text) - (to - from))
		if len(r) > room {
			r = r[:prefixClusters(r, max(room, 0))]
		}
	}
	if from == to && len(r) == 0 {
		return false
	}
	e.text = slices.Replace(e.text, from, to, r...)
	e.caret = from + len(r)
	e.anchor = e.caret
	e.hasGoal = false
	return true
}

// prefixClusters is how many runes of r's whole clusters fit in n.
func prefixClusters(r []rune, n int) int {
	fit := 0
	for _, stop := range stops(r) {
		if stop > n {
			break
		}
		fit = stop
	}
	return fit
}

func (e *Editor) save() {
	e.undo = append(e.undo, state{slices.Clone(e.text), e.caret, e.anchor})
	if len(e.undo) > 200 {
		e.undo = e.undo[1:]
	}
	e.redo = nil
}

// Do performs action, the selection extended when extend, laid out by l
// (nil for one line) and through clip for copy, cut and paste. It reports
// whether the text changed.
func (e *Editor) Do(a Action, extend bool, l Layout, clip Clipboard) bool {
	if l == nil {
		l = oneLine{e}
	}
	switch a {
	case Left, Right, WordLeft, WordRight, LineStart, LineEnd, Start, End:
		e.Place(e.horizontal(a, extend, l), extend)
	case Up, Down, PageUp, PageDown:
		e.vertical(a, extend, l)
	case SelectAll:
		e.Select(0, len(e.text))
	case Copy:
		if !e.Password && clip != nil && e.Selected() != "" {
			clip.SetClipboard(e.Selected())
		}
	case Cut:
		if e.Password || e.ReadOnly || clip == nil || e.Selected() == "" {
			return false
		}
		clip.SetClipboard(e.Selected())
		e.save()
		return e.replace(nil)
	case Paste:
		if clip == nil || e.ReadOnly {
			return false
		}
		e.save()
		e.typing = false
		return e.replace([]rune(e.clean(clip.Clipboard())))
	case Backspace, Delete, DeleteWord, DeleteToLineStart:
		return e.delete(a, l)
	case Undo:
		return e.step(&e.undo, &e.redo)
	case Redo:
		return e.step(&e.redo, &e.undo)
	case Tab:
		if e.Multiline {
			return e.Type("\t")
		}
	case Newline:
		if e.Multiline {
			e.typing = false
			return e.Type("\n")
		}
	case None, Submit:
	}
	return false
}

// horizontal is where the horizontal move a goes: Left and Right, not
// extending, collapse a selection to its start or end; the rest go from
// the caret.
func (e *Editor) horizontal(a Action, extend bool, l Layout) int {
	from, to := e.Selection()
	switch {
	case !extend && from != to && a == Left:
		return from
	case !extend && from != to && a == Right:
		return to
	}
	return e.target(a, l)
}

// target is where a horizontal move goes from the caret.
func (e *Editor) target(a Action, l Layout) int {
	switch a {
	case Left:
		return e.before(e.caret)
	case Right:
		return e.after(e.caret)
	case WordLeft:
		return wordStart(e.text, e.caret)
	case WordRight:
		return wordEnd(e.text, e.caret)
	case LineStart:
		start, _ := l.Span(l.Line(e.caret))
		return start
	case LineEnd:
		_, end := l.Span(l.Line(e.caret))
		return end
	case Start:
		return 0
	case None, End, Up, Down, PageUp, PageDown, SelectAll, Backspace, Delete, DeleteWord, DeleteToLineStart, Copy, Cut,
		Paste, Undo, Redo, Tab, Newline, Submit: // not horizontal moves; End is the text's end below
	}
	return len(e.text)
}

// lines is how many lines the vertical move a goes, page lines a page;
// 0 for any other action.
func lines(a Action, page int) int {
	switch a {
	case Up:
		return -1
	case Down:
		return 1
	case PageUp:
		return -page
	case PageDown:
		return page
	case None, Left, Right, WordLeft, WordRight, LineStart, LineEnd, Start, End, SelectAll, Backspace, Delete,
		DeleteWord, DeleteToLineStart, Copy, Cut, Paste, Undo, Redo, Tab, Newline, Submit: // not vertical moves
	}
	return 0
}

// vertical moves by visual lines, keeping the column the caret had.
func (e *Editor) vertical(a Action, extend bool, l Layout) {
	if !e.hasGoal {
		e.goal = l.X(e.caret)
	}
	goal := e.goal
	page := max(l.Page(), 1)
	n := lines(a, page)
	line := l.Line(e.caret) + n
	switch {
	case line < 0:
		e.Place(0, extend)
	case line >= l.Lines():
		e.Place(len(e.text), extend)
	default:
		e.Place(l.At(line, goal), extend)
	}
	e.goal, e.hasGoal = goal, true
}

func (e *Editor) delete(a Action, l Layout) bool {
	if e.ReadOnly {
		return false
	}
	if from, to := e.Selection(); from != to {
		e.save()
		return e.replace(nil)
	}
	var from, to int
	switch a {
	case Backspace:
		from, to = e.before(e.caret), e.caret
	case Delete:
		from, to = e.caret, e.after(e.caret)
	case DeleteWord:
		from, to = wordStart(e.text, e.caret), e.caret
	case DeleteToLineStart:
		start, _ := l.Span(l.Line(e.caret))
		from, to = start, e.caret
		if from == to {
			from = e.before(e.caret)
		}
	case None, Left, Right, WordLeft, WordRight, LineStart, LineEnd, Start, End, Up, Down, PageUp, PageDown, SelectAll,
		Copy, Cut, Paste, Undo, Redo, Tab, Newline, Submit: // only the deletions come here
	}
	if from == to {
		return false
	}
	e.save()
	e.Select(from, to)
	return e.replace(nil)
}

func (e *Editor) step(from, to *[]state) bool {
	if len(*from) == 0 {
		return false
	}
	s := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, state{slices.Clone(e.text), e.caret, e.anchor})
	e.text, e.caret, e.anchor = s.text, s.caret, s.anchor
	e.typing, e.hasGoal = false, false
	return true
}

// before and after are the cluster boundaries around i.
func (e *Editor) before(i int) int {
	prev := 0
	for _, s := range stops(e.text) {
		if s >= i {
			break
		}
		prev = s
	}
	return prev
}

func (e *Editor) after(i int) int {
	for _, s := range stops(e.text) {
		if s > i {
			return s
		}
	}
	return len(e.text)
}

// Word is the word around i, or the space or mark there, for a double
// click.
func (e *Editor) Word(i int) (int, int) {
	at := 0
	state, s := -1, string(e.text)
	for len(s) > 0 {
		var w string
		w, s, state = uniseg.FirstWordInString(s, state)
		n := utf8.RuneCountInString(w)
		if i < at+n || len(s) == 0 {
			return at, at + n
		}
		at += n
	}
	return i, i
}

// Paragraph is the line i is on, between line breaks, for a triple
// click.
func (e *Editor) Paragraph(i int) (int, int) {
	from, to := e.fit(i), e.fit(i)
	for from > 0 && e.text[from-1] != '\n' {
		from--
	}
	for to < len(e.text) && e.text[to] != '\n' {
		to++
	}
	return from, to
}

// stops are where r's grapheme clusters start, and its end.
func stops(r []rune) []int {
	out := []int{0}
	s, state, at := string(r), -1, 0
	for len(s) > 0 {
		var c string
		c, s, _, state = uniseg.FirstGraphemeClusterInString(s, state)
		at += utf8.RuneCountInString(c)
		out = append(out, at)
	}
	return out
}

// words are r's word segments as rune spans, each with whether it is a
// word, letters or digits, rather than spaces or marks.
func words(r []rune) [][3]int {
	var out [][3]int
	state, s, at := -1, string(r), 0
	for len(s) > 0 {
		var w string
		w, s, state = uniseg.FirstWordInString(s, state)
		n := utf8.RuneCountInString(w)
		isWord := 0
		for _, c := range w {
			if unicode.IsLetter(c) || unicode.IsDigit(c) {
				isWord = 1
				break
			}
		}
		out = append(out, [3]int{at, at + n, isWord})
		at += n
	}
	return out
}

// wordStart is the start of the word before i, over spaces and marks.
func wordStart(r []rune, i int) int {
	ws := words(r)
	for k := len(ws) - 1; k >= 0; k-- {
		if ws[k][0] < i && ws[k][2] == 1 {
			return ws[k][0]
		}
	}
	return 0
}

// wordEnd is the end of the word after i, over spaces and marks.
func wordEnd(r []rune, i int) int {
	for _, w := range words(r) {
		if w[1] > i && w[2] == 1 {
			return w[1]
		}
	}
	return len(r)
}

// oneLine is the layout of a one-line text when the caller gives none.
type oneLine struct{ e *Editor }

func (o oneLine) Lines() int              { return 1 }
func (o oneLine) Line(int) int            { return 0 }
func (o oneLine) Span(int) (int, int)     { return 0, len(o.e.text) }
func (o oneLine) X(i int) float32         { return float32(i) }
func (o oneLine) At(_ int, x float32) int { return o.e.fit(int(x)) }
func (o oneLine) Page() int               { return 1 }

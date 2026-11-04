package textedit_test

import (
	"testing"

	"github.com/merlindorin/ember/internal/textedit"

	"github.com/merlindorin/ember/input"
)

type clip struct{ text string }

func (c *clip) Clipboard() string                           { return c.text }
func (c *clip) SetClipboard(s string)                       { c.text = s }
func do(e *textedit.Editor, a textedit.Action, extend bool) { e.Do(a, extend, nil, nil) }

func TestEachPlatformsKeysMeanTheirMoves(t *testing.T) {
	type chord struct {
		key  input.Key
		mods textedit.Mods
	}
	mac := map[chord]textedit.Action{
		{input.KeyLeft, textedit.Mods{}}:                      textedit.Left,
		{input.KeyLeft, textedit.Mods{Alt: true}}:             textedit.WordLeft,
		{input.KeyRight, textedit.Mods{Alt: true}}:            textedit.WordRight,
		{input.KeyLeft, textedit.Mods{Super: true}}:           textedit.LineStart,
		{input.KeyRight, textedit.Mods{Super: true}}:          textedit.LineEnd,
		{input.KeyUp, textedit.Mods{Super: true}}:             textedit.Start,
		{input.KeyDown, textedit.Mods{Super: true}}:           textedit.End,
		{input.KeyHome, textedit.Mods{}}:                      textedit.LineStart,
		{input.KeyEnd, textedit.Mods{}}:                       textedit.LineEnd,
		{input.KeyA, textedit.Mods{Super: true}}:              textedit.SelectAll,
		{input.KeyC, textedit.Mods{Super: true}}:              textedit.Copy,
		{input.KeyX, textedit.Mods{Super: true}}:              textedit.Cut,
		{input.KeyV, textedit.Mods{Super: true}}:              textedit.Paste,
		{input.KeyZ, textedit.Mods{Super: true}}:              textedit.Undo,
		{input.KeyZ, textedit.Mods{Super: true, Shift: true}}: textedit.Redo,
		{input.KeyBackspace, textedit.Mods{Alt: true}}:        textedit.DeleteWord,
		{input.KeyBackspace, textedit.Mods{Super: true}}:      textedit.DeleteToLineStart,
		{input.KeyEnter, textedit.Mods{Super: true}}:          textedit.Submit,
		{input.KeyEnter, textedit.Mods{}}:                     textedit.Newline,
		{input.KeyA, textedit.Mods{Control: true}}:            textedit.None,
		{input.KeyLeft, textedit.Mods{Control: true}}:         textedit.Left,
		{input.KeyPageDown, textedit.Mods{}}:                  textedit.PageDown,
	}
	others := map[chord]textedit.Action{
		{input.KeyLeft, textedit.Mods{Control: true}}:           textedit.WordLeft,
		{input.KeyRight, textedit.Mods{Control: true}}:          textedit.WordRight,
		{input.KeyHome, textedit.Mods{}}:                        textedit.LineStart,
		{input.KeyEnd, textedit.Mods{}}:                         textedit.LineEnd,
		{input.KeyHome, textedit.Mods{Control: true}}:           textedit.Start,
		{input.KeyEnd, textedit.Mods{Control: true}}:            textedit.End,
		{input.KeyA, textedit.Mods{Control: true}}:              textedit.SelectAll,
		{input.KeyC, textedit.Mods{Control: true}}:              textedit.Copy,
		{input.KeyX, textedit.Mods{Control: true}}:              textedit.Cut,
		{input.KeyV, textedit.Mods{Control: true}}:              textedit.Paste,
		{input.KeyZ, textedit.Mods{Control: true}}:              textedit.Undo,
		{input.KeyY, textedit.Mods{Control: true}}:              textedit.Redo,
		{input.KeyZ, textedit.Mods{Control: true, Shift: true}}: textedit.Redo,
		{input.KeyBackspace, textedit.Mods{Control: true}}:      textedit.DeleteWord,
		{input.KeyEnter, textedit.Mods{Control: true}}:          textedit.Submit,
		{input.KeyA, textedit.Mods{Super: true}}:                textedit.None,
		{input.KeyLeft, textedit.Mods{Alt: true}}:               textedit.Left,
		{input.KeyUp, textedit.Mods{}}:                          textedit.Up,
		{input.KeyDelete, textedit.Mods{}}:                      textedit.Delete,
		{input.KeyTab, textedit.Mods{}}:                         textedit.Tab,
	}
	for p, set := range map[textedit.Platform]map[chord]textedit.Action{textedit.Mac: mac, textedit.Others: others} {
		for c, want := range set {
			if got, _ := textedit.Resolve(p, c.key, c.mods); got != want {
				t.Errorf("platform %d, %v %+v: %v, want %v", p, c.key, c.mods, got, want)
			}
		}
	}
	if _, extend := textedit.Resolve(textedit.Mac, input.KeyLeft, textedit.Mods{Shift: true, Alt: true}); !extend {
		t.Error("Shift extends a move")
	}
	if _, extend := textedit.Resolve(textedit.Mac, input.KeyBackspace, textedit.Mods{Shift: true}); extend {
		t.Error("Shift does not extend an edit")
	}
}

func TestMovesGoByClusterAndWord(t *testing.T) {
	e := textedit.New("née 👨‍👩‍👧 a-b, ok")
	do(e, textedit.Start, false)
	do(e, textedit.Right, false)
	do(e, textedit.Right, false)
	if e.Caret() != 2 {
		t.Fatalf("é as one: %d", e.Caret())
	}
	do(e, textedit.WordRight, false)
	if e.Caret() != 3 {
		t.Fatalf("to the end of née: %d", e.Caret())
	}
	do(e, textedit.Right, false)
	do(e, textedit.Right, false)
	if e.Caret() != 9 {
		t.Fatalf("the family as one: %d", e.Caret())
	}
	do(e, textedit.End, false)
	do(e, textedit.WordLeft, false)
	if got := string(e.Runes()[e.Caret():]); got != "ok" {
		t.Fatalf("back over the space to the start of ok: %q", got)
	}
	do(e, textedit.WordLeft, true)
	if e.Selected() != "b, " {
		t.Fatalf("Shift selects a word back: %q", e.Selected())
	}
	do(e, textedit.Left, false)
	if e.Selected() != "" || string(e.Runes()[e.Caret():]) != "b, ok" {
		t.Fatal("left without Shift collapses the selection to its start")
	}
	from, to := e.Word(1)
	if string(e.Runes()[from:to]) != "née" {
		t.Fatalf("the word around é: %q", string(e.Runes()[from:to]))
	}
}

func TestTypingReplacesTheSelectionAndUndoesInRuns(t *testing.T) {
	e := textedit.New("hello world")
	e.Select(6, 11)
	e.Type("t")
	e.Type("h")
	e.Type("e")
	e.Type("re")
	if e.Text() != "hello there" {
		t.Fatalf("typed over the selection: %q", e.Text())
	}
	do(e, textedit.Left, false)
	e.Type("!")
	do(e, textedit.Undo, false)
	if e.Text() != "hello there" {
		t.Fatalf("a move ends a run, so undo takes the ! alone: %q", e.Text())
	}
	do(e, textedit.Undo, false)
	if e.Text() != "hello world" || e.Selected() != "world" {
		t.Fatalf("the whole typed run is one step, selection back: %q %q", e.Text(), e.Selected())
	}
	do(e, textedit.Redo, false)
	if e.Text() != "hello there" {
		t.Fatalf("redo: %q", e.Text())
	}
	do(e, textedit.End, false)
	do(e, textedit.Backspace, false)
	if e.Text() != "hello ther" {
		t.Fatal("an edit after redo works")
	}
	if e.Do(textedit.Redo, false, nil, nil) {
		t.Fatal("a new edit clears what could be redone")
	}
}

func TestDeletesGoByClusterWordAndLine(t *testing.T) {
	e := textedit.New("one two 🇫🇷")
	do(e, textedit.Backspace, false)
	if e.Text() != "one two " {
		t.Fatalf("the flag at once: %q", e.Text())
	}
	do(e, textedit.DeleteWord, false)
	if e.Text() != "one " {
		t.Fatalf("a word back, over the space: %q", e.Text())
	}
	do(e, textedit.Start, false)
	do(e, textedit.Delete, false)
	if e.Text() != "ne " {
		t.Fatal("delete forward")
	}
	do(e, textedit.End, false)
	do(e, textedit.DeleteToLineStart, false)
	if e.Text() != "" {
		t.Fatalf("to the line's start: %q", e.Text())
	}
}

func TestTheClipboardRoundTrips(t *testing.T) {
	c := &clip{}
	e := textedit.New("one\ntwo 🔥")
	do(e, textedit.SelectAll, false)
	e.Do(textedit.Cut, false, nil, c)
	if e.Text() != "" || c.text != "one two 🔥" {
		t.Fatalf("one line: the break was a space; cut empties: %q %q", e.Text(), c.text)
	}
	e.Do(textedit.Paste, false, nil, c)
	e.Do(textedit.Paste, false, nil, c)
	if e.Text() != "one two 🔥one two 🔥" {
		t.Fatalf("pasted twice: %q", e.Text())
	}
	area := textedit.NewArea("")
	c.text = "a\r\nb\tc"
	area.Do(textedit.Paste, false, nil, c)
	if area.Text() != "a\nb\tc" {
		t.Fatalf("an area keeps line breaks and tabs: %q", area.Text())
	}
}

func TestMaxLengthPasswordAndReadOnly(t *testing.T) {
	e := textedit.New("abc")
	e.MaxLength = 5
	e.Type("de👨‍👩‍👧xyz")
	if e.Text() != "abcde" {
		t.Fatalf("only what fits, whole clusters: %q", e.Text())
	}
	c := &clip{text: "kept"}
	p := textedit.New("secret")
	p.Password = true
	do(p, textedit.SelectAll, false)
	p.Do(textedit.Copy, false, nil, c)
	p.Do(textedit.Cut, false, nil, c)
	if c.text != "kept" || p.Text() != "secret" {
		t.Fatal("a password copies and cuts nothing")
	}
	r := textedit.New("read me")
	r.ReadOnly = true
	do(r, textedit.SelectAll, false)
	r.Do(textedit.Copy, false, nil, c)
	r.Type("x")
	r.Do(textedit.Paste, false, nil, c)
	do(r, textedit.Backspace, false)
	if c.text != "read me" || r.Text() != "read me" {
		t.Fatalf("read-only selects and copies, nothing else: %q %q", c.text, r.Text())
	}
}

// grid lays text out in lines of four, one unit per rune.
type grid struct{ e *textedit.Editor }

func (g grid) Lines() int     { return len([]rune(g.e.Text()))/4 + 1 }
func (g grid) Line(i int) int { return min(i/4, len([]rune(g.e.Text()))/4) }
func (g grid) Span(l int) (int, int) {
	return l * 4, min(l*4+4, len([]rune(g.e.Text())))
}
func (g grid) X(i int) float32 { return float32(i - g.Line(i)*4) }
func (g grid) At(l int, x float32) int {
	s, e := g.Span(l)
	return min(s+int(x), e)
}
func (g grid) Page() int { return 2 }

func TestUpAndDownKeepTheColumn(t *testing.T) {
	e := textedit.New("abcdefghijklmn")
	l := grid{e}
	e.Place(2, false)
	e.Do(textedit.Down, false, l, nil)
	e.Do(textedit.Down, false, l, nil)
	if e.Caret() != 10 {
		t.Fatalf("two lines down, same column: %d", e.Caret())
	}
	e.Do(textedit.Down, false, l, nil)
	e.Do(textedit.Up, true, l, nil)
	if e.Caret() != 10 || e.Selected() != "klmn" {
		t.Fatalf("down to the short last line, Shift+up again, the column kept: %d %q", e.Caret(), e.Selected())
	}
	e.Place(10, false)
	e.Do(textedit.PageUp, true, l, nil)
	if e.Caret() != 2 || e.Selected() != "cdefghij" {
		t.Fatalf("a page up, selecting: %d %q", e.Caret(), e.Selected())
	}
	e.Do(textedit.LineEnd, false, l, nil)
	if e.Caret() != 4 {
		t.Fatalf("the visual line's end: %d", e.Caret())
	}
	from, to := textedit.NewArea("ab\ncd\nef").Paragraph(4)
	if from != 3 || to != 5 {
		t.Fatalf("the line between breaks: %d %d", from, to)
	}
}

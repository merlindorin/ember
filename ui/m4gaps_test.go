package ui_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

func backs(l *live, frames int) []ui.BackPressed {
	var r world.Reader[ui.BackPressed]
	var got []ui.BackPressed
	for range frames {
		l.frames(1)
		got = append(got, slices.Collect(r.Read(l.w))...)
	}
	return got
}

func press(l *live, k input.Key) {
	l.win.Input().SetKey(k, true)
	l.frames(1)
	l.win.Input().SetKey(k, false)
}

func TestEscapeOnAFocusedElementIsBack(t *testing.T) {
	b := ui.Button(ui.Text("Resume"))
	l := liveDoc(t, b)
	l.doc.Focus(b)
	l.frames(1)
	var r world.Reader[ui.BackPressed]
	press(l, input.KeyEscape)
	got := slices.Collect(r.Read(l.w))
	l.frames(1)
	got = append(got, slices.Collect(r.Read(l.w))...)
	require.Len(t, got, 1)
	assert.Same(t, b, got[0].Node)
}

func TestThePadsEastIsBack(t *testing.T) {
	b := ui.Button(ui.Text("Resume"))
	l := liveDoc(t, b)
	in := l.win.Input()
	in.ConnectGamepad(0, "pad")
	l.doc.Focus(b)
	l.frames(1)
	in.SetGamepadButton(0, input.GamepadEast, true)
	got := backs(l, 2)
	require.Len(t, got, 1)
	assert.Same(t, b, got[0].Node)
}

func TestBackWithNothingFocusedIsTheGames(t *testing.T) {
	l := liveDoc(t, ui.Button(ui.Text("Resume")))
	var r world.Reader[ui.BackPressed]
	press(l, input.KeyEscape)
	l.frames(1)
	assert.Empty(t, slices.Collect(r.Read(l.w)))
}

func TestBackClosingAModalIsNotReported(t *testing.T) {
	ok := ui.Button(ui.Text("OK"))
	dlg := ui.Dialog(ok)
	l := liveDoc(t, dlg)
	l.doc.ShowModal(dlg)
	l.frames(2)
	var r world.Reader[ui.BackPressed]
	press(l, input.KeyEscape)
	l.frames(1)
	assert.Empty(t, slices.Collect(r.Read(l.w)))
	assert.False(t, l.doc.Open(dlg))
}

func TestTheFirstAutofocusTakesTheFocusOnce(t *testing.T) {
	a := ui.Button(ui.Text("Continue"))
	b := ui.Button(ui.Text("New game"), ui.Attr("autofocus", ""))
	c := ui.Button(ui.Text("Quit"), ui.Attr("autofocus", ""))
	l := liveDoc(t, a, b, c)
	assert.True(t, l.doc.Focused(b), "the first autofocus in tree order")

	l.doc.Focus(a)
	l.frames(3)
	assert.True(t, l.doc.Focused(a), "it doesn't take the focus back once left")
}

func TestAnAutofocusAddedLaterTakesTheFocusWhenNothingHasIt(t *testing.T) {
	l := liveDoc(t, ui.Button(ui.Text("Back")))
	b := ui.Button(ui.Text("Retry"), ui.Attr("autofocus", ""))
	l.doc.Body().Append(b)
	l.frames(2)
	assert.True(t, l.doc.Focused(b))
}

func TestTheDocumentMarksTheLastDevice(t *testing.T) {
	l := liveDoc(t, ui.Button(ui.Text("Play")))
	html := l.doc.Body().Parent()
	in := l.win.Input()
	in.ConnectGamepad(0, "pad")
	in.SetGamepadButton(0, input.GamepadSouth, true)
	l.frames(2)
	v, _ := html.Attr("data-input-device")
	assert.Equal(t, "gamepad", v)

	in.SetGamepadButton(0, input.GamepadSouth, false)
	press(l, input.KeyA)
	l.frames(1)
	v, _ = html.Attr("data-input-device")
	assert.Equal(t, "keyboard", v)
}

func TestATextAreaIsNamedByItsPlaceholderNotItsText(t *testing.T) {
	area := ui.Textarea(ui.Placeholder("Notes"), ui.Text("Bring the torch"))
	l := liveDoc(t, area)
	_, all := l.auto.Snapshot("", "")
	var ids []string
	for _, e := range all {
		ids = append(ids, e.ID)
	}
	assert.Contains(t, ids, "notes")
}

func TestAKeyPressedAsTheDocumentShowsIsNotItsOwn(t *testing.T) {
	b := ui.Button(ui.Text("Resume"))
	l := liveDoc(t, b)
	l.doc.Focus(b)
	l.frames(1)
	l.doc.SetHidden(true)
	l.frames(2)

	l.doc.SetHidden(false)
	l.win.Input().SetKey(input.KeyEscape, true)
	got := backs(l, 1)
	l.win.Input().SetKey(input.KeyEscape, false)
	got = append(got, backs(l, 2)...)
	assert.Empty(t, got, "the Escape that showed it went to the game")

	l.win.Input().SetKey(input.KeyEscape, true)
	got = backs(l, 2)
	require.Len(t, got, 1, "the next Escape is the document's")
	assert.Same(t, b, got[0].Node)
}

func TestAFieldTheGameSetsKeepsTypingButTakesItsAnswerToASubmit(t *testing.T) {
	say := ui.Input(ui.Type("text"))
	l := liveDoc(t, say)
	l.doc.Focus(say)
	l.frames(1)
	in := l.win.Input()
	for _, r := range "hello" {
		in.AddText(r)
	}
	l.frames(1)
	l.doc.SetValue(say, "pushed")
	assert.Equal(t, "hello", l.doc.Value(say), "what is being typed stays")

	in.SetKey(input.KeyEnter, true)
	l.frames(1)
	in.SetKey(input.KeyEnter, false)
	l.frames(1)
	require.NotEmpty(t, l.submits)
	require.True(t, l.doc.Focused(say))
	l.doc.SetValue(say, "")
	assert.Equal(t, "", l.doc.Value(say), "a submitted field takes the game's answer, still focused")

	in.AddText('a')
	l.frames(1)
	l.doc.SetValue(say, "pushed")
	assert.Equal(t, "a", l.doc.Value(say), "typing again is kept again")
}

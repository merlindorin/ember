package control_test

import (
	"testing"

	"github.com/lantervale/ember/ui/control"

	"github.com/lantervale/ember/internal/textedit"
	"github.com/stretchr/testify/assert"
)

type clip struct{ s string }

func (c *clip) Clipboard() string     { return c.s }
func (c *clip) SetClipboard(s string) { c.s = s }

func TestTypingIntoATextInputIsInputAndChangedComesOnCommit(t *testing.T) {
	in := control.NewText("")
	in.Focus()
	assert.Equal(t, control.Input, in.Type("ab"))
	assert.Equal(t, "ab", in.Value())
	assert.Equal(t, control.Submitted|control.Changed, in.Key(textedit.Newline, false, nil))
	assert.Equal(t, control.Submitted, in.Key(textedit.Newline, false, nil), "submitted again, nothing new to change")
	in.Type("c")
	assert.Equal(t, control.Changed, in.Blur())
	in.Focus()
	assert.Equal(t, control.Events(0), in.Blur(), "no edit since focus")
}

func TestATextInputEditsThroughTheSharedEditor(t *testing.T) {
	in := control.NewText("hello world")
	c := &clip{}
	in.Focus()
	assert.Equal(t, control.Events(0), in.Key(textedit.SelectAll, false, c))
	assert.Equal(t, control.Events(0), in.Key(textedit.Copy, false, c))
	assert.Equal(t, "hello world", c.s)
	assert.Equal(t, control.Input, in.Key(textedit.Backspace, false, c))
	assert.Equal(t, "", in.Value())
	assert.Equal(t, control.Input, in.Key(textedit.Undo, false, c))
	assert.Equal(t, "hello world", in.Value())
	assert.Equal(t, control.Events(0), in.Key(textedit.Left, true, c), "moving is not input")
}

func TestATextInputHonoursMaxLengthAndReadOnly(t *testing.T) {
	in := control.NewText("")
	in.SetMaxLength(3)
	in.Type("abcdef")
	assert.Equal(t, "abc", in.Value())

	ro := control.NewText("fixed")
	ro.SetReadOnly(true)
	assert.Equal(t, control.Events(0), ro.Type("x"))
	assert.Equal(t, "fixed", ro.Value())
}

func TestASetValueIsNoEvent(t *testing.T) {
	in := control.NewText("a")
	in.Focus()
	in.SetValue("b")
	assert.Equal(t, "b", in.Value())
	assert.Equal(t, control.Events(0), in.Blur(), "a script's value is not the user's change")
}

func TestASearchInputClearsOnBack(t *testing.T) {
	s := control.NewSearch("query")
	s.Focus()
	assert.True(t, s.Handles(control.Back))
	assert.Equal(t, control.Input, s.Do(control.Back))
	assert.Equal(t, "", s.Value())
	assert.False(t, s.Handles(control.Back), "empty, Back goes up to the focus")
}

func TestAPasswordShowsBulletsAndNeverCopies(t *testing.T) {
	p := control.NewPassword("secret")
	assert.Equal(t, "••••••", p.Shown())
	c := &clip{s: "kept"}
	p.Key(textedit.SelectAll, false, c)
	p.Key(textedit.Copy, false, c)
	assert.Equal(t, "kept", c.s)
	assert.Equal(t, "secret", p.Value())
}

func TestATextAreaTakesNewlinesAndSubmitsByShortcut(t *testing.T) {
	a := control.NewTextArea("")
	a.Focus()
	a.Type("one")
	assert.Equal(t, control.Input, a.Key(textedit.Newline, false, nil))
	a.Type("two")
	assert.Equal(t, "one\ntwo", a.Value())
	assert.Equal(t, control.Submitted, a.Key(textedit.Submit, false, nil))
	assert.Equal(t, control.Changed, a.Blur())
}

func TestANumberInputStepsByArrowsAndKeepsOnlyNumberCharacters(t *testing.T) {
	n := control.NewNumber("5")
	n.Min, n.Max, n.Step = 0, 10, 2
	n.Focus()
	assert.True(t, n.Handles(control.Up))
	assert.Equal(t, control.Input, n.Do(control.Up))
	assert.Equal(t, "6", n.Value(), "steps from Min: 5 goes up to the next step")
	n.Do(control.Up)
	n.Do(control.Up)
	n.Do(control.Up)
	assert.Equal(t, "10", n.Value())
	assert.Equal(t, control.Events(0), n.Do(control.Up))
	n.Do(control.Down)
	assert.Equal(t, "8", n.Value())

	n.SetValue("")
	n.Type("1a2.5e-x")
	assert.Equal(t, "12.5e-", n.Value())
	_, ok := n.Number()
	assert.False(t, ok, "not a number yet")
	n.Key(textedit.SelectAll, false, nil)
	n.Type("2.5")
	v, ok := n.Number()
	assert.True(t, ok)
	assert.Equal(t, 2.5, v)
	assert.Equal(t, control.Changed, n.Blur())
}

func TestANumberInputWithoutBoundsStepsFreely(t *testing.T) {
	n := control.NewNumber("")
	n.Do(control.Down)
	assert.Equal(t, "-1", n.Value(), "empty counts as 0")
	n.SetValue("0.1")
	n.Step = 0.1
	n.Do(control.Up)
	assert.Equal(t, "0.2", n.Value(), "no float noise")
}

func TestANumberSettlesInRangeOrBackToItsValue(t *testing.T) {
	n := control.NewNumber("10")
	n.Min, n.Max, n.Step = 0, 100, 5
	n.Editor.SetText("1000")
	n.Settle()
	assert.Equal(t, "100", n.Value(), "kept within max")
	assert.Equal(t, control.Changed, n.Blur(), "then committed")
	n.Editor.SetText("12")
	n.Settle()
	assert.Equal(t, "10", n.Value(), "on a step")
	n.Editor.SetText("-")
	n.Settle()
	assert.Equal(t, "100", n.Value(), "what isn't a number goes back to the committed value")
}

func TestShownIsMadeOnlyWhenTheTextChanges(t *testing.T) {
	in := control.NewText("Heron")
	first := in.Shown()
	assert.Zero(t, testing.AllocsPerRun(10, func() { in.Shown() }))
	in.Editor.SetText("Crane")
	assert.Equal(t, "Crane", in.Shown())
	assert.Equal(t, "Heron", first)
	pw := control.NewPassword("abc")
	assert.Equal(t, "•••", pw.Shown())
	pw.Editor.SetText("abcd")
	assert.Equal(t, "••••", pw.Shown())
}

func TestEmptyReadsTheTextWithoutAString(t *testing.T) {
	in := control.NewText("")
	assert.True(t, in.Empty())
	in.Editor.SetText("a")
	assert.False(t, in.Empty())
	assert.Zero(t, testing.AllocsPerRun(10, func() { in.Empty() }))
}

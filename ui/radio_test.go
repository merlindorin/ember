package ui_test

import (
	"slices"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// changedIn is the Changed events f's frames emit.
func changedIn(l *live, f func()) []*ui.Node {
	from := len(l.changes)
	f()
	return l.changes[from:]
}

func radio(name, value string, args ...ui.Arg) *ui.Node {
	return ui.Input(append([]ui.Arg{ui.Type("radio"), ui.Attr("name", name), ui.Value(value)}, args...)...)
}

func TestARadioGroupKeepsOneCheckedAndTheArrowsMoveIt(t *testing.T) {
	a, b := radio("size", "s"), radio("size", "m", ui.Attr("checked", ""))
	off, c := radio("size", "l", ui.Disabled()), radio("size", "xl")
	other := radio("shape", "round", ui.Attr("checked", ""))
	before := ui.Button(ui.Text("Before"))
	l := liveDoc(t, before, a, b, off, c, other)
	require.True(t, l.doc.Checked(b))

	changed := changedIn(l, func() { l.clickAt(a) })
	assert.True(t, l.doc.Checked(a))
	assert.False(t, l.doc.Checked(b), "the group's other radio unchecks")
	assert.True(t, l.doc.Checked(other), "another name is another group")
	assert.True(t, slices.Equal([]*ui.Node{a}, changed))
	assert.Equal(t, "s", l.doc.Value(a))

	l.key(input.KeyRight)
	assert.True(t, l.doc.Checked(b) && l.doc.Focused(b), "an arrow moves the check and the focus")
	l.key(input.KeyRight)
	assert.True(t, l.doc.Checked(c), "the disabled one is skipped")
	l.key(input.KeyDown)
	assert.True(t, l.doc.Checked(a), "it wraps")
	l.key(input.KeyUp)
	assert.True(t, l.doc.Checked(c), "and back")

	l.doc.Focus(before)
	l.frames(1)
	l.key(input.KeyTab)
	assert.True(t, l.doc.Focused(c), "Tab reaches the group at its checked radio")
	l.key(input.KeyTab)
	assert.True(t, l.doc.Focused(other), "and leaves it for the next group")
}

func TestAnAriaRadioGroupWorksTheSame(t *testing.T) {
	x := ui.Button(ui.Attr("role", "radio"), ui.Attr("aria-checked", "true"), ui.Text("Tiles"))
	y := ui.Button(ui.Attr("role", "radio"), ui.Attr("aria-checked", "false"), ui.Text("Objects"))
	z := ui.Button(ui.Attr("role", "radio"), ui.Attr("aria-checked", "false"), ui.Text("Lights"))
	group := ui.Div(ui.Attr("role", "radiogroup"), x, y, z)
	l := liveDoc(t, group)

	changed := changedIn(l, func() { l.clickAt(z) })
	assert.True(t, slices.Equal([]*ui.Node{z}, changed))
	v, _ := z.Attr("aria-checked")
	assert.Equal(t, "true", v)
	v, _ = x.Attr("aria-checked")
	assert.Equal(t, "false", v)
	assert.Contains(t, l.clicks, z, "a button radio is still clicked")

	l.key(input.KeyLeft)
	assert.True(t, l.doc.Checked(y) && l.doc.Focused(y))

	var labels []string
	var on []any
	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		if e.Role == automation.RoleSelect {
			labels = append(labels, e.Label)
			on = append(on, e.Value)
		}
	}
	assert.Equal(t, []string{"Tiles", "Objects", "Lights"}, labels)
	assert.Equal(t, []any{false, true, false}, on)
}

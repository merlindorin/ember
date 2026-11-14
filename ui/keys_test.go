package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

// keyed records the KeyPressed events of the frames run.
type keyed struct {
	l   *live
	r   world.Reader[ui.KeyPressed]
	got []ui.KeyPressed
}

func (k *keyed) frames(n int) {
	for range n {
		k.l.frames(1)
		for e := range k.r.Read(k.l.w) {
			k.got = append(k.got, e)
		}
	}
}

// typed presses key with text typed alongside it, and releases it.
func (k *keyed) typed(key input.Key, text string) {
	in := k.l.win.Input()
	in.SetKey(key, true)
	for _, r := range text {
		in.AddText(r)
	}
	k.frames(1)
	in.SetKey(key, false)
	k.frames(2)
}

func TestAKeyNoControlTakesIsPressedOnTheFocusedElement(t *testing.T) {
	b := ui.Button(ui.Text("Save"))
	l := liveDoc(t, b)
	k := &keyed{l: l}
	l.doc.Focus(b)
	k.frames(1)
	k.typed(input.KeyF, "f")
	require.Len(t, k.got, 1)
	got := k.got[0]
	assert.Same(t, b, got.Node)
	got.Node = nil
	assert.Equal(t, ui.KeyPressed{Key: input.KeyF, Text: "f"}, got)
}

func TestTypingInATextFieldPressesNothing(t *testing.T) {
	field := ui.Input()
	l := liveDoc(t, field)
	k := &keyed{l: l}
	l.doc.Focus(field)
	k.frames(1)
	k.typed(input.KeyF, "f")
	assert.Empty(t, k.got)
	assert.Equal(t, "f", l.doc.Value(field))
}

func TestAGridTakesItsArrowsButNotCommandUp(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	grid, _, _, rows := files()
	l := liveDoc(t, grid)
	k := &keyed{l: l}
	l.clickAt(rows[1])
	k.typed(input.KeyDown, "")
	assert.Empty(t, k.got, "the grid took Down")
	in := l.win.Input()
	in.SetKey(input.KeyLeftSuper, true)
	k.frames(1)
	k.typed(input.KeyUp, "")
	in.SetKey(input.KeyLeftSuper, false)
	k.frames(1)
	require.Len(t, k.got, 1)
	got := k.got[0]
	assert.Same(t, grid, got.Node)
	got.Node = nil
	assert.Equal(t, ui.KeyPressed{Key: input.KeyUp, Command: true}, got)
}

func TestAKeyInAModalWithNothingFocusedIsTheDialogs(t *testing.T) {
	d := ui.Dialog(ui.Div(ui.Text("note")))
	l := liveDoc(t, d)
	k := &keyed{l: l}
	l.doc.ShowModal(d)
	k.frames(2)
	k.typed(input.KeyQ, "q")
	require.Len(t, k.got, 1)
	assert.Same(t, d, k.got[0].Node)
}

func TestWithNothingFocusedTheKeysAreTheGames(t *testing.T) {
	l := liveDoc(t, ui.Button(ui.Text("Save")))
	k := &keyed{l: l}
	k.typed(input.KeyW, "w")
	assert.Empty(t, k.got)
}

func TestAStillFrameWithAFocusedButtonAllocatesNothingForKeys(t *testing.T) {
	b := ui.Button(ui.Text("Save"))
	l := liveDoc(t, b)
	l.doc.Focus(b)
	l.frames(3)
	in := l.win.Input()
	allocs := testing.AllocsPerRun(20, func() { ui.Keys(l.doc, in) })
	assert.Zero(t, allocs)
}

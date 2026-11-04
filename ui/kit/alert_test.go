package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnAlertIsAnsweredByItsButtonsOrEscape(t *testing.T) {
	dont := kit.Button("Don't save", ui.Class("kit-destructive"))
	cancel := kit.Button("Cancel")
	save := kit.Button("Save", ui.Class("kit-primary"), ui.Value("save"))
	a := kit.Alert("Save the tileset?", "Your changes will be lost otherwise.", dont, cancel, save)
	behind := kit.Button("Behind")
	s := show(t, behind, a)
	closed := watch[ui.Closed](s)
	require.Zero(t, a.Box().W, "hidden until shown")

	s.doc.ShowModal(a)
	s.frames(2)
	assert.Greater(t, a.Box().W, float32(380-1))
	assert.True(t, s.doc.Focused(save), "the primary button takes the focus")
	assert.Less(t, dont.Box().X, cancel.Box().X, "a destructive button stands on the left")
	assert.Equal(t, automation.RoleDialog, s.element(t, "Save the tileset?").Role)

	s.click(behind)
	assert.Empty(t, *closed, "the page behind is inert")
	s.key(input.KeyEnter)
	require.Len(t, *closed, 1)
	assert.Same(t, a, (*closed)[0].Node)
	assert.Equal(t, "save", (*closed)[0].Value)

	s.doc.ShowModal(a)
	s.frames(2)
	s.click(dont)
	assert.Equal(t, "Don't save", (*closed)[1].Value, "a button's text is its value unless set")

	s.doc.ShowModal(a)
	s.frames(2)
	s.key(input.KeyEscape)
	assert.Equal(t, "", (*closed)[2].Value, "Escape cancels")
}

func TestADestructiveAlertFocusesItsSafeButtonAndIsNamedByItsTitle(t *testing.T) {
	keep := kit.Button("Keep")
	replace := kit.Button("Replace", ui.Class("kit-destructive"))
	a := kit.Alert("Replace it?", "It overwrites what it holds.", keep, replace)
	s := show(t, a)
	s.doc.ShowModal(a)
	s.frames(2)
	assert.True(t, s.doc.Focused(keep), "without a primary, the first button that isn't destructive takes the focus")
	assert.Less(t, replace.Box().X, keep.Box().X, "the destructive one stays on the left")
	assert.Equal(t, "replace-it", s.element(t, "Replace it?").ID)
}

func TestSetAlertAsksAgainAboutSomethingElse(t *testing.T) {
	ok, cancel := kit.Button("OK", ui.Class("kit-primary")), kit.Button("Cancel")
	a := kit.Alert("Save walls?", "Lost otherwise.", cancel, ok)
	s := show(t, a)
	kit.SetAlert(a, "Save floor?", "Still lost otherwise.")
	s.doc.ShowModal(a)
	s.frames(2)
	s.element(t, "Save floor?")
	assert.Equal(t, "Save floor?", a.Children()[1].Text())
	assert.Equal(t, "Still lost otherwise.", a.Children()[2].Text())

	s.doc.Focus(cancel)
	s.frames(1)
	kit.SetAlert(a, "Save props?", "Lost too.")
	s.frames(2)
	assert.True(t, s.doc.Focused(cancel), "changed while open, the alert keeps its focus")
	assert.Equal(t, "save-props", s.element(t, "Save props?").ID)
}

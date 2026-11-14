package ui_test

import (
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
)

func TestAResetButtonEmptiesItsFieldAndShowsOnlyWhileASearchHoldsText(t *testing.T) {
	field := ui.Input(ui.ID("q"), ui.Type("search"), ui.Value("crypt"))
	reset := ui.Button(ui.Type("reset"), ui.Attr("aria-controls", "q"), ui.Attr("aria-label", "clear"),
		ui.Style("width: 14px; height: 14px"))
	l := liveDoc(t, field, reset)
	assert.Greater(t, reset.Box().W, float32(0), "shown while the search holds text")
	assert.Contains(t, l.roles(automation.RoleButton), "clear", "named by its aria-label")

	l.doc.Focus(field)
	l.frames(1)
	l.clickAt(reset)
	assert.Equal(t, "", l.doc.Value(field))
	assert.Contains(t, l.changes, field, "the field reports the change")
	assert.True(t, l.doc.Focused(field), "the focus stays in the field")
	assert.Zero(t, reset.Box().W, "hidden once the search is empty")

	l.typed("w")
	l.frames(1)
	assert.Greater(t, reset.Box().W, float32(0), "shown again once something is typed")
}

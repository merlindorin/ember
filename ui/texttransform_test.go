package ui_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/ui"
	"github.com/stretchr/testify/assert"
)

func TestTextTransformShowsCasedTextButKeepsItsSource(t *testing.T) {
	save := ui.Button(ui.Style("text-transform: uppercase"), ui.Text("Save tileset"))
	field := ui.Input(ui.Type("text"), ui.Style("text-transform: uppercase"), ui.Value("straße"))
	l := liveDoc(t, save, field)
	assert.Contains(t, l.roles(automation.RoleButton), "Save tileset", "automation reads the source")
	assert.Equal(t, "Save tileset", save.Text())
	assert.Equal(t, "straße", l.doc.Value(field), "an edit field's value is as typed")
}

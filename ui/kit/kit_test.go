package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestTheThemeParsesWithoutWarnings(t *testing.T) {
	d := ui.NewDocument(kit.Theme())
	d.Body().Append(ui.P(ui.Text("x")))
	d.Layout(geom.V2(200, 100))
	assert.Empty(t, d.Warnings())
	assert.Len(t, d.Sheets(), 1)
}

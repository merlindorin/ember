package kit_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestTheThemeParsesWithoutWarnings(t *testing.T) {
	d := ui.NewDocument(kit.Theme())
	d.Body().Append(ui.P(ui.Text("x")))
	d.Layout(geom.V2(200, 100))
	assert.Empty(t, d.Warnings())
	assert.Len(t, d.Sheets(), 1)
}

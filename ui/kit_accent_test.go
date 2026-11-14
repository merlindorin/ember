package ui_test

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestPlainControlsInAKitDocumentTakeKitsAccent(t *testing.T) {
	box := ui.Input(ui.Type("checkbox"), ui.Checked(true))
	sw := ui.Input(ui.Type("checkbox"), ui.Attr("switch", ""), ui.Checked(true))
	r := ui.Input(ui.Type("range"))
	bar := ui.Progress(ui.Attr("value", "0.5"))
	d := ui.NewDocument(kit.Theme())
	d.Body().Append(box, sw, r, bar)
	d.Layout(geom.V2(400, 300))
	accent := color.Hex("#ffce76")
	assert.Equal(t, accent, ui.ShownBackground(box), "a checked checkbox")
	assert.Equal(t, accent, ui.ShownBackground(sw), "a switch on")
	assert.Equal(t, accent, ui.ShownAccent(r), "a range's fill")
	assert.Equal(t, accent, ui.ShownAccent(bar), "a progress bar's fill")
	assert.Equal(t, color.RGBA(0, 0, 0, 0.7), ui.ShownBackground(bar), "on kit's dark track, not a faint blue")
}

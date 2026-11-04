package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestALabelStaysOnOneLineUnlessItWraps(t *testing.T) {
	d := ui.NewDocument(kit.Theme())
	long := "The quick brown fox jumps over the lazy dog"
	one := kit.Label(long)
	wrapped := kit.Label(long, ui.Class("kit-wrap"))
	cut := kit.Label(long, ui.Class("kit-truncate"))
	d.Body().Append(ui.Div(ui.Style("width: 120px"), one), ui.Div(ui.Style("width: 120px"), wrapped),
		ui.Div(ui.Style("width: 120px"), cut))
	d.Layout(geom.V2(400, 300))

	assert.Greater(t, one.Box().W, float32(120), "one line, past its box")
	assert.Equal(t, float32(120), wrapped.Box().W)
	assert.Greater(t, wrapped.Box().H, 2*one.Box().H, "several lines")
	assert.Equal(t, float32(120), cut.Box().W)
	assert.Equal(t, one.Box().H, cut.Box().H, "one line, cut")
	assert.Equal(t, long, cut.Text(), "the text is whole; only its drawing is cut")
	assert.Empty(t, d.Warnings())
}

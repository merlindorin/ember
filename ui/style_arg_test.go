package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

func TestAStyleArgumentKeepsWhatEarlierArgumentsDeclared(t *testing.T) {
	child := ui.Div(ui.Style("height: 10px"))
	p := kit.Panel(kit.Padding(24), ui.Style("width: 380px"), ui.Style("padding-top: 4px"), child)
	assert.Contains(t, p.Style(), "padding: 24px")
	assert.Contains(t, p.Style(), "width: 380px")

	d := ui.NewDocument()
	d.Body().Append(p)
	d.Layout(geom.V2(640, 400))
	assert.InDelta(t, 24, child.Box().X-p.Box().X, 1.5, "the padding stays beside the width")
	assert.InDelta(t, 4, child.Box().Y-p.Box().Y, 1.5, "a later declaration wins")
}

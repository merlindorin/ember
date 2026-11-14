package ui_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

const shrinkToFit = "position: absolute; left: 20px; top: 40px; display: flex; flex-direction: column; min-width: 120px"

// scrollingPanel is a box as wide as its content, positioned, holding a
// body of twenty lines that scrolls once the box is held short.
func scrollingPanel(style string) (*ui.Document, *ui.Node) {
	var lines []ui.Arg
	for i := range 20 {
		lines = append(lines,
			ui.P(ui.Style("margin: 0; white-space: nowrap"), ui.Text(fmt.Sprintf("line %d of the panel", i))))
	}
	body := ui.Div(append([]ui.Arg{ui.Style("overflow-y: auto; flex: 1; min-height: 0")}, lines...)...)
	panel := ui.Div(ui.Style(style), body)
	d := ui.NewDocument()
	d.Body().Append(panel)
	return d, panel
}

func TestAShrinkToFitBoxTakesItsBodysNewScrollbarInAtOnce(t *testing.T) {
	d, panel := scrollingPanel(shrinkToFit)
	d.Layout(geom.V2(640, 360))
	wide := panel.Box().W
	panel.SetStyle(shrinkToFit + "; max-height: 120px")
	d.Layout(geom.V2(640, 360))
	assert.InDelta(t, wide+15, panel.Box().W, 0.01, "held short, it widens by its body's scrollbar in the same layout")

	f, fresh := scrollingPanel(shrinkToFit + "; max-height: 120px")
	f.Layout(geom.V2(640, 360))
	assert.Equal(t, fresh.Box().W, panel.Box().W, "as wide as one laid out short from the start")
}

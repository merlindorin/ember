//go:build chrome

package kit_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/kit"
)

func TestTextWidthsRoundUpToChromesLayoutUnit(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			d := ui.NewDocument(kit.Theme())
			d.Body().Append(
				kit.Button("Save"), kit.Button("Delete", ui.Class("kit-primary")), kit.Label("default"),
				ui.P(ui.Style("width: 230px; text-align: right"), ui.Text("The quick brown fox "),
					ui.Span(ui.Style("padding: 0 3px"), ui.Text("jumps")), ui.Text(" over the lazy dog, then naps.")),
				ui.Div(ui.Style("display: inline-block"), ui.Text("shrink to fit")),
			)
			r := chrome.MatchScaled(t, d, geom.V2(640, 400), scale)
			assert.Less(t, r.MaxDelta, 1/(64*scale), "every edge within one layout unit")
		})
	}
}

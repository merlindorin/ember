package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
)

func TestANodeMovedToAnotherDocumentKeepsTheBoxItsNewOneGaveIt(t *testing.T) {
	from, to := ui.NewDocument(), ui.NewDocument()
	moved := ui.Div(ui.Style("width: 50px; height: 20px"))
	from.Body().Append(moved)
	from.Layout(geom.V2(200, 100))
	from.Body().Remove(moved)
	to.Body().Append(moved)
	to.Layout(geom.V2(200, 100))
	from.Layout(geom.V2(200, 100))
	assert.Equal(t, float32(50), moved.Box().W, "the old document laying out after leaves it alone")
}

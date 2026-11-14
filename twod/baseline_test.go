package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaselineTakesTheOriginAsTheTextsBaseline(t *testing.T) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	assert.True(t, NewText("x", Baseline()).Baseline())
	assert.False(t, NewText("x").Baseline())

	n := len(c.list.Vertices)
	c.Text(geom.V2(10, 30), "x", FontSize(16), Baseline())
	require.Len(t, c.list.Vertices, n+4, "one glyph")
	bottom := c.list.Vertices[n+2].Pos.Y
	assert.InDelta(t, 30, bottom, 0.5, "x stands on the origin")
	assert.InDelta(t, 10, c.list.Vertices[n].Pos.X, 1.5, "at the origin's x, give or take the glyph's bearing")
}

func TestABaselineTextEntityIgnoresItsVerticalPivot(t *testing.T) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	n := len(c.list.Vertices)
	NewText("x", Baseline(), FontSize(16)).Draw(c)
	require.Len(t, c.list.Vertices, n+4)
	assert.InDelta(t, 0, c.list.Vertices[n+2].Pos.Y, 0.5, "centred across, but standing on its origin")
}

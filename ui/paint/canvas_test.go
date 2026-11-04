package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestATreePaintsOntoAWindowsCanvas(t *testing.T) {
	w := world.New()
	win := platform.NewWindow(platform.Size(320, 200))
	w.AddEntities(win)
	w.AddPlugins(twod.Plugin{})
	w.Frame(1.0 / 60)

	card := el(20, 20, 100, 60, func(s *style.Computed) {
		s.Background = red
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	})
	card.Append(text(4, 4, "hi", card.Style))
	c := twod.CanvasFor(w, win)
	require.NotNil(t, c)
	Paint(c, tree(el(0, 0, 320, 200, nil, card)))

	l := draw.FrontFor(w, win)
	require.NotNil(t, l)
	require.NotEmpty(t, l.Vertices)
	assert.Equal(t, geom.V2(20, 20), l.Vertices[0].Pos, "the card's background first, in window points")
	assert.Equal(t, red, l.Vertices[0].Color)
	assert.Greater(t, len(l.Vertices), 4, "and its text's glyphs")
}

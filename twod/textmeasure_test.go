package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"slices"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
)

func TestCanvasTextReturnsItsMeasureAndDrawsAsBefore(t *testing.T) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	for _, text := range []string{"Handgloves", "tile 12 · Edit", "x", ""} {
		for _, opts := range [][]TextOption{
			{FontSize(15)},
			{FontSize(15), Baseline()},
			{FontSize(13), Pivot(0, 0), Tracking(0.5)},
			{FontSize(16), Baseline(), TextShadow(color.Black)},
			{FontSize(16), Pivot(0.5, 0.5)},
			{FontSize(16), Pivot(0, 1), Baseline()},
		} {
			want := c.Measure(text, opts...)
			n := len(c.list.Vertices)
			assert.Equal(t, want, c.Text(geom.V2(10.3, 30), text, opts...), "%q", text)
			drawn := slices.Clone(c.list.Vertices[n:])

			tx := c.scratchText(text, opts)
			size := tx.style().Measure(tx.shown(text))
			off := geom.V2(size.X*tx.pivot.X, size.Y*tx.pivot.Y)
			if tx.baseline {
				off.Y = 0
			}
			n = len(c.list.Vertices)
			tx.paint(c, geom.V2(10.3, 30).Sub(off), tx.shown(text))
			assert.Equal(t, c.list.Vertices[n:], drawn, "%q: the same quads as measuring first", text)
		}
	}
}

func BenchmarkCanvasText(b *testing.B) {
	w, _, _ := app()
	w.RunFrames(1)
	c := CanvasFor(w, nil)
	opts := []TextOption{FontSize(15), Baseline(), Color(color.White)}
	b.ReportAllocs()
	for range b.N {
		c.list.Vertices, c.list.Indices, c.list.Batches = c.list.Vertices[:0], c.list.Indices[:0], c.list.Batches[:0]
		c.Text(geom.V2(10, 20), "tile 412 · walls · Edit", opts...)
	}
}

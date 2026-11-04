package layout_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/fonts"
	"github.com/merlindorin/ember/ui/layout"
)

// BenchmarkInlineText lays out fifty wrapped paragraphs with spans, in the Go font, as each
// frame does.
func BenchmarkInlineText(b *testing.B) {
	const words = "The quick brown fox jumps over the lazy dog, then naps in the warm afternoon sun."
	d := ui.NewDocument()
	for range 50 {
		d.Body().Append(ui.P(ui.Style("width: 240px"), ui.Text(words+" "),
			ui.Span(ui.Style("font-weight: bold"), ui.Text("Bold part")), ui.Text(" "+words)))
	}
	size := geom.V2(800, 600)
	d.Layout(size)
	fs := fonts.Of(d)
	b.ReportAllocs()
	for b.Loop() {
		layout.Layout(d.HTML(), size, fs)
	}
}

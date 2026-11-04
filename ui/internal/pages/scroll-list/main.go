// A 1,000-row list in a kit.Scroll filling the window, the rows kit's
// BenchmarkScrolling builds (its scroller 400 points high), to scroll for
// paint and GPU measurements, captures and GIFs.
package main

import (
	"fmt"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

func main() {
	d := ui.NewDocument(kit.Theme())
	d.HTML().SetStyle("background-color: var(--window-color)")
	col := kit.Column()
	for i := range 1000 {
		col.Append(kit.Row(kit.Label(fmt.Sprintf("tile %d", i)), kit.Button("Edit")))
	}
	d.Body().Append(kit.Scroll(ui.Style("height: 600px; background-color: var(--window-color)"), col))
	pages.Run(d, 800, 600)
}

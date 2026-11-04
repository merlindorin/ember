// A list of 1,000 elements: rows of a label and a button in a kit.Column, as
// kit's BenchmarkScale builds it, for paint and GPU measurements and captures.
package main

import (
	"fmt"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

func main() {
	d := ui.NewDocument(kit.Theme())
	col := kit.Column(ui.Style("background-color: var(--window-color)"))
	for i := range 1000 / 3 {
		col.Append(kit.Row(kit.Label(fmt.Sprintf("tile %d", i)), kit.Button("Edit")))
	}
	d.Body().Append(col)
	pages.Run(d, 800, 600)
}

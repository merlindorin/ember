// A table of 1,000 elements: rows of three cells, as kit's BenchmarkScale builds
// it, for paint and GPU measurements and captures.
package main

import (
	"fmt"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

func main() {
	d := ui.NewDocument(kit.Theme())
	body := ui.Tbody()
	for i := range 1000 / 4 {
		body.Append(ui.Tr(ui.Td(ui.Text(fmt.Sprintf("tile %d", i))), ui.Td(ui.Text("walls")), ui.Td(ui.Text("true"))))
	}
	d.Body().Append(ui.Table(ui.Style("background-color: var(--window-color); color: var(--light-text)"), body))
	pages.Run(d, 800, 600)
}

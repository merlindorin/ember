// The M0 page: a bordered card with a paragraph and a button.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

func main() {
	d := ui.NewDocument()
	d.Body().Append(
		ui.Div(ui.ID("card"), ui.Style("padding: 12px; border: 2px solid #335"),
			ui.P(ui.Text("Hello, ember")),
			ui.Button(ui.Style("padding: 4px; border: 1px solid #888"), ui.Text("Play")),
		),
	)
	pages.Run(d, 800, 600)
}

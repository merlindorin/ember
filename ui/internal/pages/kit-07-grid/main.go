// 07 Grid: kit.Grid with two and three columns, gaps, padding and custom tracks, on
// ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 260px;
  gap: 12px 16px; align-items: baseline; }
.frame { border: 1px dashed #5a524a; }
.cell { background-color: var(--button-color); border: 1px solid var(--border-color); border-radius: 3px;
  padding: 2px 6px; }
`

func cell(text string) *ui.Node { return ui.Div(ui.Class("cell"), ui.Text(text)) }

func cells(texts ...string) []ui.Arg {
	args := make([]ui.Arg, len(texts))
	for i, t := range texts {
		args[i] = cell(t)
	}
	return args
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, g *ui.Node) {
		g.AddClass("frame")
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), g)
	}
	row("2 columns", kit.Grid(2, cells("name", "grass", "solid", "true", "light radius", "64")...))
	row("3 columns", kit.Grid(3, cells("a", "bb", "ccc", "dddd", "e", "ff", "g")...))
	row("gap 10", kit.Grid(2, append([]ui.Arg{kit.Gap(10)}, cells("one", "two", "three", "four")...)...))
	row("padding 8", kit.Grid(2, append([]ui.Arg{kit.Padding(8)}, cells("one", "two", "three", "four")...)...))
	row("max-content 1fr", kit.Grid(2, append([]ui.Arg{kit.Columns("max-content 1fr")}, cells("name", "grass",
		"occluder group", "walls")...)...))
	row("1fr 1fr", kit.Grid(2, append([]ui.Arg{kit.Columns("1fr 1fr")}, cells("left", "right", "a longer cell",
		"b")...)...))
	pages.Run(d, 420, 480)
}

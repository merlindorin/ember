// 03 Icons: kit.NewIcon and kit.Icon at every size and colour on ember's window colour, in
// The old ember/ui's test icon font (a square and a triangle).
package main

import (
	"embed"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

//go:embed icons.ttf names.json
var files embed.FS

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 200px;
  gap: 10px 16px; align-items: baseline; }
.box { border: 1px dashed #5a524a; }
`

func main() {
	set, err := kit.LoadIcons(files, "icons.ttf", "names.json")
	if err != nil {
		panic(err)
	}
	d := ui.NewDocument(kit.Theme(), set.Sheet(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}},
		"page.css"))
	row := func(name string, n *ui.Node) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), ui.Div(ui.Class("box"), n))
	}
	row("icon", kit.NewIcon(set, "square"))
	row("13px", kit.NewIcon(set, "triangle", ui.Style("font-size: 13px")))
	row("20px", kit.NewIcon(set, "triangle", ui.Style("font-size: 20px")))
	row("30px", kit.NewIcon(set, "square", ui.Style("font-size: 30px")))
	row("accent", kit.NewIcon(set, "square", ui.Style("color: var(--accent-color)")))
	row("muted", kit.NewIcon(set, "triangle", ui.Style("color: var(--muted-color)")))
	row("label", kit.Label("Save", kit.Icon(set, "triangle")))
	row("caps", kit.Label("TILES", ui.Class("kit-caps"), kit.Icon(set, "square")))
	row("rich text", kit.RichText("[b]Bold[/b] after", kit.Icon(set, "square")))
	row("missing", kit.Label("no icon", kit.Icon(set, "circle")))
	pages.Run(d, 340, 400)
}

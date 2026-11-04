// 13 Progress: kit.Progress empty, part full and full, without its
// value, and over a Range, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px 24px; display: grid;
  grid-template-columns: max-content max-content; gap: 14px 24px; align-items: center; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, p *ui.Node) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), p)
	}
	row("0", kit.Progress(0))
	row("30", kit.Progress(30))
	row("100", kit.Progress(100))
	row("no value", kit.Progress(65, kit.ShowValue(false)))
	row("range 10–210 at 70", kit.Progress(70, kit.Range(10, 210)))
	pages.Run(d, 400, 230)
}

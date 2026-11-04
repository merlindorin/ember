// 27 StatusBar: kit.StatusBar in window footers, saved, unsaved with its dot and an
// element added, and cut where its text runs long, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.window-footer { width: 340px; padding: 6px 10px; border-top: 1px solid var(--window-edge);
  background-color: var(--window-bar); }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	footer := func(bar *ui.Node) *ui.Node { return ui.Div(ui.Class("window-footer"), bar) }
	edited := kit.StatusBar("walls.tsj · 41 tiles · 3 shapes", kit.Label("100%", ui.Class("kit-caps")))
	kit.SetDot(edited, true)
	long := kit.StatusBar("rooms-of-the-long-forgotten-ossuary.tsj · 412 tiles · 38 shapes · edited")
	kit.SetDot(long, true)
	d.Body().Append(
		footer(kit.StatusBar("props.tsj · 42 tiles")),
		footer(edited),
		footer(long),
	)
	pages.Run(d, 380, 150)
}

// 19 PathPopUp: kit.PathPopUp closed, cut at its width, and held open on its folders,
// beside a path cut at its start by class truncate-start, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: flex; flex-direction: column;
  align-items: flex-start; gap: 12px; }
.path { width: 200px; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; color: var(--muted-color); }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	long := kit.PathPopUp("/Users/ada/Projects/Hearthstead/go/game-ui/engine/demo/crypt/tiled/" +
		"rooms-of-the-long-forgotten-ossuary")
	open := kit.PathPopUp("/Users/ada/assets/survivors/crypt")
	d.Body().Append(
		kit.PathPopUp("/Users/ada/assets/survivors/crypt/tiled"),
		long,
		kit.Label("/Users/ada/assets/survivors/crypt/tiled/walls.tsj", ui.Class("path", "truncate-start")),
		kit.Label("/short/path", ui.Class("path", "truncate-start")),
		open,
	)
	menu := open.Children()[len(open.Children())-1]
	menu.SetAttr("open", "")
	open.SetAttr("aria-expanded", "true")
	pages.Pin(d, menu.Children()[1], dom.Hover)
	pages.Run(d, 480, 340)
}

// 29 MenuBar: a kit.MenuBar with its File menu held open, and one with Edit hovered, on
// ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `body { background-color: var(--window-color); display: flex; flex-direction: column; gap: 170px; }`

func bar() (*ui.Node, *ui.Node, *ui.Node, *ui.Node) {
	fileMenu := kit.Menu(
		kit.MenuItem("New tileset", kit.Shortcut("Cmd+N")),
		kit.MenuItem("Open…", kit.Shortcut("Cmd+O")),
		kit.Separator(),
		kit.MenuItem("Save", kit.Shortcut("Cmd+S")),
		kit.MenuItem("Save as…", kit.Shortcut("Cmd+Shift+S")),
		kit.Separator(),
		kit.MenuItem("Quit", kit.Shortcut("Cmd+Q")),
	)
	file := kit.MenuBarItem("File", fileMenu)
	edit := kit.MenuBarItem("Edit", kit.Menu(kit.MenuItem("Undo"), kit.MenuItem("Redo")))
	b := kit.MenuBar(file, edit, kit.MenuBarItem("View", kit.Menu(kit.MenuItem("Grid"))), kit.MenuBarItem("Help",
		kit.Menu(kit.MenuItem("About"))))
	return b, file, fileMenu, edit
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	open, file, fileMenu, _ := bar()
	hovered, _, _, edit := bar()
	d.Body().Append(open, hovered)
	fileMenu.SetAttr("open", "")
	file.SetAttr("aria-expanded", "true")
	pages.Pin(d, edit, dom.Hover)
	pages.Run(d, 420, 260)
}

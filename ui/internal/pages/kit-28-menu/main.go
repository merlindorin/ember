// 28 Menu: a kit.MenuButton's menu held open, with shortcuts, a separator, a disabled item,
// checkable items on and off, a hovered item and an open submenu, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `body { background-color: var(--window-color); padding: 16px; }`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	rename := kit.MenuItem("Rename…", kit.Shortcut("F2"))
	recent := kit.SubMenu("Move to", kit.MenuItem("Walls"), kit.MenuItem("Floors"), kit.MenuItem("Props"))
	menu := kit.Menu(
		kit.MenuItem("Duplicate", kit.Shortcut("Cmd+D")),
		rename,
		recent,
		kit.Separator(),
		kit.CheckMenuItem("Pixel grid", true),
		kit.CheckMenuItem("Shadows", false),
		kit.Separator(),
		kit.MenuItem("Paste style", ui.Attr("aria-disabled", "true")),
		kit.MenuItem("Delete", kit.Shortcut("Del")),
	)
	button := kit.MenuButton("Tile actions", menu)
	d.Body().Append(button)
	menu.SetAttr("open", "")
	button.SetAttr("aria-expanded", "true")
	sub := recent.Children()[2]
	sub.SetAttr("open", "")
	recent.SetAttr("aria-expanded", "true")
	pages.Pin(d, rename, dom.Hover)
	pages.Run(d, 420, 320)
}

// 26 Thumbnails: kit.Thumbnails with a selected cell, a dimmed one, a long name and one
// without a picture, then smaller cells with the focus ring, on ember's window colour;
// the picture is the 24×24 panel of example 22.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

//go:embed panel.png
var panel []byte

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: flex; flex-direction: column; gap: 10px; }
.files { height: 248px; }
.small { height: 106px; }
`

func main() {
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}, "panel.png": {Data: panel}}
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(files, "page.css"))
	cells := []*ui.Node{
		kit.Thumbnail("walls.tsj", "panel.png"),
		kit.Thumbnail("floor.tsj", "panel.png"),
		kit.Thumbnail("a very long tileset name.tsj", "panel.png"),
		kit.Thumbnail("props.tsj", "panel.png"),
		kit.Thumbnail("old.tsj", "panel.png"),
		kit.Thumbnail("no picture", ""),
	}
	kit.SetDimmed(cells[4], true)
	grid := kit.Thumbnails(ui.Class("files"))
	grid.Append(cells...)
	small := kit.Thumbnails(ui.Class("small"), kit.ThumbnailSize(84, 92),
		kit.Thumbnail("crypt", "panel.png"), kit.Thumbnail("sanctuary", "panel.png"), kit.Thumbnail("lava",
			"panel.png"), kit.Thumbnail("ice", "panel.png"))
	d.Body().Append(kit.Label("selected, dimmed, a long name, no picture", ui.Class("kit-form-label")), grid,
		kit.Label("ThumbnailSize(84, 92), focused", ui.Class("kit-form-label")), small)
	d.SetSelected(grid, 1)
	d.SetSelected(small, 2)
	pages.Pin(d, small, dom.Focus|dom.FocusVisible)
	pages.Run(d, 420, 500)
}

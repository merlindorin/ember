// 24c Table spans: a kit.Table under a caption, its Tileset title over two
// columns, a tileset's cell spanning its two rows, and a footer row
// totalling the sizes.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; }
.kit-table { width: 380px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	cell := func(text string, attrs ...string) *ui.Node {
		args := []ui.Arg{ui.Text(text)}
		for i := 0; i+1 < len(attrs); i += 2 {
			args = append(args, ui.Attr(attrs[i], attrs[i+1]))
		}
		return ui.Td(args...)
	}
	rows := []*ui.Node{
		ui.Tr(cell("walls.tsj", "rowspan", "2"), cell("collision"), cell("24 KB")),
		ui.Tr(cell("light"), cell("6 KB")),
		ui.Tr(cell("floor.tsj"), cell("picture"), cell("9 KB")),
		ui.Tr(cell("props.tsj"), cell("objects"), cell("12 KB")),
	}
	d.Body().Append(kit.Table([]*ui.Node{kit.TableColumn("Tileset", ui.Attr("colspan", "2")), kit.TableColumn("Size",
		kit.ColumnWidth(70), kit.ColumnAlign("right"))}, rows,
		ui.Caption(ui.Text("Tilesets in the crypt")), ui.Tfoot(ui.Tr(cell("Total", "colspan", "2"), cell("51 KB")))))
	pages.Run(d, 412, 220)
}

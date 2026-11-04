// 24b Table, scrolled: a 60-row kit.Table scrolled part way down, its titles stuck above
// the rows, row 25 selected just below them, on ember's window colour.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; }
.kit-table { width: 380px; height: 200px; }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	var rows []*ui.Node
	for i := range 60 {
		rows = append(rows, kit.TableRow(fmt.Sprintf("tile_%02d.png", i), fmt.Sprintf("%d KB", 3+i*7%90), "Picture"))
	}
	table := kit.Table([]*ui.Node{kit.TableColumn("Name", kit.Sortable()), kit.TableColumn("Size", kit.ColumnWidth(70),
		kit.ColumnAlign("right")), kit.TableColumn("Kind", kit.ColumnWidth(90))}, rows)
	d.Body().Append(table)
	kit.SortedBy(d, table, 0, false)
	d.SetSelected(kit.TableGrid(table), 25)
	d.Layout(geom.V2(412, 232))
	head := kit.TableGrid(table).Children()[0].Box()
	dom.Data(table).Scroll = geom.V2(0, rows[25].Box().Y-head.Y-head.H)
	pages.Run(d, 412, 232)
}

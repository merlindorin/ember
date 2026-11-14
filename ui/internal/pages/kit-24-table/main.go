// 24 Table: kit.Table of files by Name, Size and Kind, sorted by name with its arrow up,
// walls.tsj selected, a row hovered and a folder dimmed; and a smaller one sorted by size,
// its arrow down, on ember's window colour.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.kit-table { width: 380px; }
.files { height: 196px; }
`

func files() []*ui.Node {
	var rows []*ui.Node
	for _, r := range [][3]string{
		{"crypt", "—", "Folder"},
		{"floor.png", "120 KB", "Picture"},
		{"props.tsj", "9 KB", "Tileset"},
		{"README.txt", "2 KB", "Text"},
		{"rooms.tmj", "61 KB", "Map"},
		{"walls.tsj", "24 KB", "Tileset"},
	} {
		rows = append(rows, kit.TableRow(r[0], r[1], r[2]))
	}
	return rows
}

func columns() []*ui.Node {
	return []*ui.Node{kit.TableColumn("Name", kit.Sortable()), kit.TableColumn("Size", kit.ColumnWidth(70),
		kit.ColumnAlign("right"), kit.Sortable()), kit.TableColumn("Kind", kit.ColumnWidth(90))}
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	rows := files()
	kit.SetDimmed(rows[0], true)
	byName := kit.Table(columns(), rows, ui.Class("files"))
	bySize := kit.Table(columns(), files()[1:4])
	for i, r := range kit.TableRows(bySize) {
		kit.SortValue(r, 1, []string{"120", "9", "2"}[i])
	}
	d.Body().Append(byName, bySize)
	kit.SortedBy(d, byName, 0, false)
	kit.SortedBy(d, bySize, 1, true)
	d.SetSelected(kit.TableGrid(byName), 5)
	pages.Pin(d, rows[2], dom.Hover)
	pages.Run(d, 412, 340)
}

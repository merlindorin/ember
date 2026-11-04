# Table

![A table of files sorted by name, walls.tsj selected and a folder dimmed, and a smaller one sorted by size, descending](table.png "ember · table")

<!-- chrome -->
Matches Chrome within 0.40% of its pixels (threshold 0.52%).
<!-- /chrome -->

```go
files := kit.Table([]*ui.Node{
	kit.TableColumn("Name", kit.Sortable()),
	kit.TableColumn("Size", kit.ColumnWidth(70), kit.ColumnAlign("right"), kit.Sortable()),
	kit.TableColumn("Kind", kit.ColumnWidth(90)),
}, rows)
row := kit.TableRow("walls.tsj", "24 KB", "Tileset")
```

| Option | Default | |
|---|---|---|
| `kit.Sortable()` | not sortable | a press on the title sorts by its column, then reversed, with an arrow |
| `kit.ColumnWidth(70)` | an equal share | the column at least 70 points |
| `kit.ColumnAlign(kit.Right())` | left | the title and its cells aligned: `kit.Right()` or `kit.Center()` |
| `kit.NoHeader()` on the table | titles shown | no titles |

```go
d.Selected(kit.TableGrid(files))        // the selected row, -1 for none
d.SetSelected(kit.TableGrid(files), 3)  // no event
kit.SortedBy(d, files, 1, true)         // by Size, descending, no event
kit.SortValue(row, 1, "24")             // sorts by 24, not "24 KB"
kit.SetDimmed(row, true)                // greyed, never selected
kit.CellOf(row, 0)
kit.TableRows(files)                    // in the order they show
```

| Event | When |
|---|---|
| `ui.Changed{Node}` | a row selected by a click or Up, Down, Home, End, `Node` the `TableGrid` |
| `ui.Submitted{Node}` | the selected row activated: Enter, Space, the pad's South, a double click |
| `ui.Clicked{Node}` | a `Sortable` title pressed, `Node` the title |

### Scrolled

![A 60-row table scrolled part way, its titles stuck above row 25, selected](table-scrolled.png "ember · table-scrolled")

<!-- chrome 24b-table-scrolled -->
Matches Chrome within 0.57% of its pixels (threshold 0.75%).
<!-- /chrome -->

### Spans, caption and footer

![A table of tilesets under its caption, Tileset over two columns, walls.tsj spanning two rows, and a Total footer](table-spans.png "ember · table spans")

<!-- chrome 24c-table-spans -->
<!-- /chrome -->

```go
t := kit.Table([]*ui.Node{kit.TableColumn("Tileset", ui.Attr("colspan", "2")), kit.TableColumn("Size")}, []*ui.Node{
	ui.Tr(ui.Td(ui.Text("walls.tsj"), ui.Attr("rowspan", "2")), ui.Td(ui.Text("collision")), ui.Td(ui.Text("24 KB"))),
	ui.Tr(ui.Td(ui.Text("light")), ui.Td(ui.Text("6 KB"))),
},
	ui.Caption(ui.Text("Tilesets")),
	ui.Tfoot(ui.Tr(ui.Td(ui.Text("Total"), ui.Attr("colspan", "2")), ui.Td(ui.Text("30 KB")))),
)
kit.CellOf(row, 2) // the cell in column 2, or the one spanning it
```

A title or a cell spans with `colspan` and `rowspan`; a sort title past a spanning one sorts its own column, and a column's alignment follows them. A `ui.Caption` or `ui.Tfoot` among `kit.Table`'s arguments goes into the table: the caption over it, `kit.Below()` on the caption for under it, the footer under the rows, its cells aligned as their columns, spans included, and never selectable.

| CSS | |
|---|---|
| `.kit-table` | the scrolling box, `:focus-visible` on its `table` |
| `.kit-table th`, `.kit-table td` | titles and cells; `th` stick to the top while scrolling |
| `.kit-right`, `.kit-center` | a column's alignment, on its `th` and `td` |
| `.kit-table tbody > tr` | rows: striped, `:hover`, `[aria-selected="true"]`, `[aria-disabled="true"]` |
| `.kit-table th[aria-sort]` | a sortable title, its arrow on `ascending` and `descending` |
| `.kit-table.kit-no-header` | titles hidden |
| `.kit-table caption`, `.kit-table caption.kit-below` | the caption, 12 points, ruled off from the titles, or from the rows under them |
| `.kit-table tfoot td` | footer cells, ruled off and muted |

Automation: titles by their `id`, else their slug (`size`), `press`; rows by their `id`, else the slug of their first cell (`walls-tsj`), value whether selected; `press` selects and activates, `select` only selects.

```
go run ./examples/92-ui-kit -element table
```

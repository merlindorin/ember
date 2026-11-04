# table

![A score table under its caption: Player over both header rows, Run over Score and Time, three rows and a Total footer](table.png "ember · table")

```go
scores := ui.Table(ui.Attr("role", "grid"),
	ui.Caption(ui.Text("Arena, week 12")),
	ui.Thead(
		ui.Tr(ui.Th(ui.Text("Player"), ui.Attr("rowspan", "2")), ui.Th(ui.Text("Run"), ui.Attr("colspan", "2"))),
		ui.Tr(ui.Th(ui.Text("Score"), ui.Attr("aria-sort", "descending")), ui.Th(ui.Text("Time"), ui.Attr("aria-sort", "none"))),
	),
	ui.Tbody(
		ui.Tr(ui.Td(ui.Text("Ayla")), ui.Td(ui.Text("4210")), ui.Td(ui.Text("12:40"))),
		ui.Tr(ui.Td(ui.Text("Bram")), ui.Td(ui.Text("3875")), ui.Td(ui.Text("11:02"))),
	),
	ui.Tfoot(ui.Tr(ui.Td(ui.Text("Total")), ui.Td(ui.Text("8085")), ui.Td(ui.Text("23:42")))),
)
doc.Sort(scores, 1, true) // as a press on Score does, without events
doc.SortBy(scores, func(a, b *ui.Node, column int, descending bool) int { … })
doc.Selected(scores)      // the selected body row, -1 for none
```

Laid out by CSS's automatic table layout: columns as wide as their cells, rows as tall as their tallest cell, a cell spanning several sharing its extra width over its columns by their content and its height over its rows by theirs. With `table-layout: fixed` and a width, the first row's sized cells set their columns, the rest share what's left, and later rows widen nothing. Its children are a `caption` (above, or below with `caption-side: bottom`), `thead`, `tbody`, `tfoot` (under the bodies wherever it's written) or `tr`; only the first `thead` and `tfoot` are header and footer, others count as bodies. The table's border, padding, background and shadow frame the grid, not the caption; its outline goes round both. A selected row and a sorted header have no default look, as in Chrome: `kit.Table` styles them.

| Attribute | |
|---|---|
| `role="grid"` | one Tab stop; a click, Up, Down, Home or End selects a body row (`aria-selected="true"`); Enter, Space, the pad's South or a double click activates it |
| `aria-sort` on a `th` | a press sorts the body rows by its column, ascending then reversed: by each cell's `data-sort`, else its text, as numbers when both read as one, else ignoring case |
| `aria-disabled="true"` on a row | dimmed and skipped |
| `colspan`, `rowspan` on a `th` or `td` | the cell spans columns and rows ([th](33-th.md)) |

| Default CSS | |
|---|---|
| `table` | `border-spacing: 2px` |

| Event | When |
|---|---|
| `ui.Changed{Node}` | a grid's row selected |
| `ui.Submitted{Node}` | a grid's selection activated |
| `ui.Sorted{Node, Column, Descending}` | a grid sorted by a press on a header, `aria-sort` already set |

Automation: a grid's body rows are recorded by the slug of their first cell (`ayla`), `press` and `select`; its sorting headers as buttons, `press`.

```
go run ./examples/93-html-elements -element table
```

# tfoot

![The score table's Total row under its three players](table.png "ember · table")

```go
ui.Table(thead, tbody, ui.Tfoot(ui.Tr(ui.Td(ui.Text("Total")), ui.Td(ui.Text("11035")), ui.Td(ui.Text("33:00")))))
```

The table's footer rows: laid out under every body wherever it's written. Only a table's first `tfoot` is its footer; another counts as a body. In a grid, its rows aren't selected or sorted.

| Default CSS | |
|---|---|
| `tfoot` | none: laid out as a row group |

```
go run ./examples/93-html-elements -element tfoot
```

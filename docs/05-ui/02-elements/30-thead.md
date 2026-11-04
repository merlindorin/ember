# thead

![A score table: the header row Player, Score and Time in bold](table.png "ember · table")

```go
ui.Table(
	ui.Thead(ui.Tr(ui.Th(ui.Text("Player")), ui.Th(ui.Text("Score")))),
	ui.Tbody(…),
)
```

Its rows are the table's head: in a grid, never selected or sorted.

| Default CSS | |
|---|---|
| `thead` | none |

```
go run ./examples/93-html-elements -element thead
```

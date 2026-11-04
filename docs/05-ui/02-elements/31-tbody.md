# tbody

![A score table: the body rows Ayla, Bram and Corin under the header](table.png "ember · table")

```go
ui.Table(
	ui.Thead(…),
	ui.Tbody(
		ui.Tr(ui.Td(ui.Text("Ayla")), ui.Td(ui.Text("4210"))),
		ui.Tr(ui.Td(ui.Text("Bram")), ui.Td(ui.Text("3875"))),
	),
)
```

Its rows are the table's body: in a grid, the rows a player selects and a header sorts.

| Default CSS | |
|---|---|
| `tbody` | none |

```
go run ./examples/93-html-elements -element tbody
```

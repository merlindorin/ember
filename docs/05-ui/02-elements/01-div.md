# div

![A hero's stats, three lines in a dashed box](div.png "ember · div")

```go
ui.Div(ui.Style("border: 1px dashed #999; padding: 8px"),
	ui.Div(ui.Text("Health 80 / 100")),
	ui.Div(ui.Text("Mana 35 / 50")),
	ui.Div(ui.Text("Gold 120")),
)
```

| Default CSS | |
|---|---|
| `div` | `display: block` |

A box with no meaning: for grouping and styling when no other block says what the content is.

Automation: not recorded, unless it has an `id` (then a label of its text), as every block; its `id` or `data-name` prefixes its descendants' IDs.

```
go run ./examples/93-html-elements -element div
```

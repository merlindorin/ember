# header

![An inventory's title and slot count above a dashed line, its content below](header.png "ember · header")

```go
ui.Header(
	ui.H2(ui.Text("Inventory")),
	ui.P(ui.Text("12 of 20 slots used")),
)
```

| Default CSS | |
|---|---|
| `header` | `display: block` |

Laid out as a div: it says the content introduces its screen or section (a title, a summary, its tools).

Automation: not recorded, unless it has an `id` (then a label of its text), as every block; no landmark role yet; its `id` or `data-name` prefixes its descendants' IDs.

```
go run ./examples/93-html-elements -element header
```

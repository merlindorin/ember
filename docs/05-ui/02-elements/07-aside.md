# aside

![A shop's goods, a tip in a grey dashed box beside them](aside.png "ember · aside")

```go
ui.Div(ui.Style("display: flex; gap: 12px"),
	ui.Div(ui.H3(ui.Text("Shop")), ui.P(ui.Text("Potion, 10 gold"))),
	ui.Aside(ui.Style("width: 160px"),
		ui.P(ui.Text("Tip: lanterns light the crypt for ten minutes.")),
	),
)
```

| Default CSS | |
|---|---|
| `aside` | `display: block` |

Laid out as a div, not floated or beside anything: it says the content is apart from what is round it (a tip, a note). The flex row puts it to the side.

Automation: not recorded, unless it has an `id` (then a label of its text), as every block; no landmark role yet; its `id` or `data-name` prefixes its descendants' IDs.

```
go run ./examples/93-html-elements -element aside
```

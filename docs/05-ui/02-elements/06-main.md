# main

![A quest log: its title, its quests in a dashed box, its keys](main.png "ember · main")

```go
ui.Main(
	ui.P(ui.Text("Find the lost key")),
	ui.P(ui.Text("Light the three beacons")),
	ui.P(ui.Text("Talk to the smith")),
)
```

| Default CSS | |
|---|---|
| `main` | `display: block` |

Laid out as a div: it says the content is the screen's own, one per page, apart from its header, footer and nav.

Automation: not recorded, unless it has an `id` (then a label of its text), as every block; no landmark role yet; its `id` or `data-name` prefixes its descendants' IDs.

```
go run ./examples/93-html-elements -element main
```

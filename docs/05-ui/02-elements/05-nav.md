# nav

![A pause menu's four buttons in a dashed box](nav.png "ember · nav")

```go
ui.Nav(ui.ID("menu"),
	ui.Button(ui.Text("Resume")),
	ui.Text(" "),
	ui.Button(ui.Text("Map")),
)
```

| Default CSS | |
|---|---|
| `nav` | `display: block` |

Laid out as a div: it says the content leads to the game's other screens.

Automation: not recorded, unless it has an `id` (then a label of its text), as every block; no landmark role yet; its `id` or `data-name` prefixes its descendants' IDs. Here the buttons are `menu/resume`, `menu/map`…

```
go run ./examples/93-html-elements -element nav
```

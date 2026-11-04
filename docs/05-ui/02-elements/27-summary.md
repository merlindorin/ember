# summary

![A closed inventory: a triangle, Inventory and 12 of 20 slots in italics](summary.png "ember · summary")

```go
ui.Details(
	ui.Summary(ui.Text("Inventory "), ui.Em(ui.Text("12 of 20 slots"))),
	ui.P(ui.Text("A rope, a lantern, three apples.")),
)
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its label; children may be any inline elements |

A click, Enter or Space while focused, or the pad's South opens or closes its details.

| Default CSS | |
|---|---|
| `summary` | `display: list-item` |
| `details > summary` | `list-style-type: disclosure-closed` (▶) |
| `details[open] > summary`, `details:open > summary` | `list-style-type: disclosure-open` (▼) |

| Event | When |
|---|---|
| `ui.Clicked{Node}` | pressed, with `ui.Toggled` for its details |

Automation: its `id`, else `data-name`, `aria-label`, then the slug of its text (`inventory-12-of-20-slots`); `press`.

```
go run ./examples/93-html-elements -element summary
```

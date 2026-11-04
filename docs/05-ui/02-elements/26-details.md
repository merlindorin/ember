# details

![A quest log: The old well open, its text below, and Lost keys closed](details.png "ember · details")

```go
well := ui.Details(ui.Open(),
	ui.Summary(ui.Text("The old well")),
	ui.P(ui.Text("Bring three buckets of water to the mill.")),
)
doc.SetOpen(well, false) // without events
doc.Open(well)           // false
```

| Attribute | |
|---|---|
| `ui.Open()` | starts it open; `open` follows its state, so `details[open]` and `details:open` match |

| Default CSS | |
|---|---|
| `details` | `display: block` |
| `details:not([open]) > :not(summary)` | `display: none`: closed, only its summary shows |

| Event | When |
|---|---|
| `ui.Toggled{Node}` | opened or closed by a press on its summary |

Automation: not recorded itself; its summary is.

```
go run ./examples/93-html-elements -element details
```

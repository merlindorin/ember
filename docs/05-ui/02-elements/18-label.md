# label

![Name, then a text field holding Ayla](label.png "ember · label")

```go
ui.Label(ui.For("name"), ui.Text("Name"))
ui.Input(ui.ID("name"), ui.Value("Ayla"))
```

| Attribute | |
|---|---|
| `ui.For(id)` | the control it is for; without it, the first control inside it |
| `ui.Text(s)` | its text; children may be any inline elements |

A click on it focuses and clicks its control, as in HTML.

| Default CSS | |
|---|---|
| `label` | none: inline, like span |

Automation: its text names the control it labels when that has no `id`, `data-name`, `aria-label` or text of its own; a label's text never includes the control it labels.

```
go run ./examples/93-html-elements -element label
```

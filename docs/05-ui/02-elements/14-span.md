# span

![Gold: 1,250, the number in a span coloured gold](span.png "ember · span")

```go
ui.P(ui.Text("Gold: "), ui.Span(ui.Style("color: #b8860b"), ui.Text("1,250")))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

| Default CSS | |
|---|---|
| `span` | none: inline; it shows only what its own `Style` or a sheet gives it |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element span
```

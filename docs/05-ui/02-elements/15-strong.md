# strong

![Press Space to jump, Space in bold](strong.png "ember · strong")

```go
ui.P(ui.Text("Press "), ui.Strong(ui.Text("Space")), ui.Text(" to jump."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

| Default CSS | |
|---|---|
| `strong` | `font-weight: bold` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element strong
```

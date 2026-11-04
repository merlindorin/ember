# kbd

![Press Ctrl+S to save, the keys in monospace](kbd.png "ember · kbd")

```go
ui.P(ui.Text("Press "), ui.Kbd(ui.Text("Ctrl")), ui.Text("+"), ui.Kbd(ui.Text("S")), ui.Text(" to save."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | the key or keys; children may be any inline elements |

| Default CSS | |
|---|---|
| `kbd` | `font-family: "Go Mono", monospace; font-size: 0.8125em`, as `code` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element kbd
```

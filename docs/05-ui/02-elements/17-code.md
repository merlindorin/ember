# code

![Type /invite name to add a player, the command in monospace](code.png "ember · code")

```go
ui.P(ui.Text("Type "), ui.Code(ui.Text("/invite name")), ui.Text(" to add a player."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

| Default CSS | |
|---|---|
| `code` | `font-family: "Go Mono", monospace; font-size: 0.8125em` (13 points in 16) |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element code
```

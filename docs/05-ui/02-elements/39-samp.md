# samp

![The console says player not found, the answer in monospace](samp.png "ember · samp")

```go
ui.P(ui.Text("The console says "), ui.Samp(ui.Text("player not found")), ui.Text("."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | what a program printed; children may be any inline elements |

| Default CSS | |
|---|---|
| `samp` | `font-family: "Go Mono", monospace; font-size: 0.8125em`, as `code` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element samp
```

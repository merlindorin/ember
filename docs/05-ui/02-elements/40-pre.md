# pre

![Three lines of a save file, their spaces kept, in monospace](pre.png "ember · pre")

```go
ui.Pre(ui.Text("level:  3\nhero:   Ayla\ngold:   1250"))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text, line breaks and spaces kept, never wrapped |

| Default CSS | |
|---|---|
| `pre` | `display: block; margin: 1em 0; white-space: pre; font-family: "Go Mono", monospace; font-size: 0.8125em` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element pre
```

# caption

![The score table's caption, Arena, week 12, centred above it](table.png "ember · table")

```go
ui.Table(ui.Caption(ui.Text("Arena, week 12")), thead, tbody)
```

Any child of the table, first by HTML's convention: laid out at the table's width, above its grid, or below with `caption-side: bottom` on the table or the caption. The table is at least as wide as its caption needs, and the table's border, padding, background and shadow stay round the grid, the caption outside them.

| Default CSS | |
|---|---|
| `caption` | `text-align: center` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element caption
```

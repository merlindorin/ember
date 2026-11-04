# footer

![An inventory with its keys below a dashed line](footer.png "ember · footer")

```go
ui.Footer(ui.Text("Enter to use, Esc to close"))
```

| Default CSS | |
|---|---|
| `footer` | `display: block` |

Laid out as a div: it says the content closes its screen or section (keys, credits, a status).

Automation: not recorded, unless it has an `id` (then a label of its text), as every block; no landmark role yet; its `id` or `data-name` prefixes its descendants' IDs.

```
go run ./examples/93-html-elements -element footer
```

# progress

![A download bar at 70%, and a waiting bar with its sliding segment](progress.png "ember · progress")

```go
ui.Label(ui.For("download"), ui.Text("Download"))
ui.Progress(ui.ID("download"), ui.Value("70"), ui.Attr("max", "100"))
ui.Progress() // no value: indeterminate
```

| Attribute | |
|---|---|
| `ui.Value(v)` | how far, from 0 to `max`; without it, a segment slides across the bar |
| `max` | the end (1) |

| Default CSS | |
|---|---|
| `progress` | `display: inline-block; width: 160px; height: 16px; color: #0075ff` |

Its `accent-color`, else `color`, fills the bar; its `background-color` paints the track, `color` faded without one.

Automation: named by its `aria-label` or `<label>`, not recorded without one; a bar whose value is `value` over `max` (`0.7`), none while indeterminate.

```
go run ./examples/93-html-elements -element progress
```

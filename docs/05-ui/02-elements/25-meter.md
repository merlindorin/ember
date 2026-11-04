# meter

![Three needs as meters: health high in green, food halfway in yellow, water low in red](meter.png "ember · meter")

```go
ui.Meter(ui.Value("0.5"),
	ui.Attr("low", "0.3"), ui.Attr("high", "0.7"), ui.Attr("optimum", "1"))
```

| Attribute | |
|---|---|
| `ui.Value(v)` | where it stands, from `min` to `max` |
| `min`, `max` | its range (0, 1) |
| `low`, `high` | where the low and high regions end (`min`, `max`) |
| `optimum` | the best value (the middle); its region is green, the next yellow, the far one red |

| Default CSS | |
|---|---|
| `meter` | `display: inline-block; width: 80px; height: 16px` |

Its `background-color` paints the track, `color` faded without one.

Automation: named by its `aria-label` or `<label>`, not recorded without one; a meter whose value is `value`.

```
go run ./examples/93-html-elements -element meter
```

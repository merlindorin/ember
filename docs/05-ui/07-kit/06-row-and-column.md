# Row and Column

![Rows and columns of chips in dashed frames: plain, a column, gap 12, padding 8, an expanding child, justified centre, end and between, stretched, aligned start, centre and end, and the toolbar, body and footer paddings](row-and-column.png "ember · row-and-column")

<!-- chrome 06-row-column -->
Matches Chrome within 0.11% of its pixels (threshold 0.26%).
<!-- /chrome -->

```go
footer := kit.Footer(kit.JustifyEnd(), cancel, apply)
body := kit.Column(kit.Body(), music, effects)
```

| Option | Default | |
|---|---|---|
| `kit.Gap(px)` | 4 | points between children |
| `kit.Padding(px)` | 0 | points inside the edge |
| `kit.JustifyStart()`, `kit.JustifyCenter()`, `kit.JustifyEnd()`, `kit.JustifyBetween()` | start | children along it |
| `kit.AlignStart()`, `kit.AlignCenter()`, `kit.AlignEnd()` | stretched | children across it |
| `kit.Toolbar(args…)` | | a Row padded 8, gap 8, centred across |
| `kit.Body()` | | padding 16, gap 8 |
| `kit.Footer(args…)` | | a Row padded 12, gap 8, centred across |
| `kit.Expand()` on a child | its size | grows into the room left |
| `kit.Fill()` on a child | | stretched across |

| CSS | |
|---|---|
| `.kit-row`, `.kit-column` | flex, left to right or top to bottom, gap 4 |
| `.kit-justify-*`, `.kit-align-*` | placement |
| `.kit-toolbar`, `.kit-body`, `.kit-footer` | the paddings |
| `.kit-row > .kit-expand`, `.kit-row > .kit-fill` (and in `.kit-column`) | the children's |

Automation: not seen; with an `id` it prefixes its children's IDs (`audio/apply`).

```
go run ./examples/92-ui-kit -element row-and-column
```

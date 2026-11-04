# Scroll

![Scroll boxes overflowing down, both ways, fitting, with the vertical bar always and never](scroll.png "ember · scroll")

<!-- chrome -->
Matches Chrome within 0.28% of its pixels (threshold 0.94%).
<!-- /chrome -->

```go
log := kit.Scroll(ui.Style("height: 200px"), lines...)
```

| Option | Default | |
|---|---|---|
| `kit.VerticalBar(kit.BarAlways)` | `kit.BarAuto` | not supported yet: shows the bar only when the content overflows, like BarAuto |
| `kit.VerticalBar(kit.BarNever)` | `kit.BarAuto` | no bar; the wheel and keys still scroll, and both axes' bars hide |
| `kit.HorizontalBar(…)` | `kit.BarAuto` | the same across |

| CSS | |
|---|---|
| `.kit-scroll` | clips and scrolls its content; may shrink below it in a row or column |

Automation: none of its own; its content is seen where it shows.

```
go run ./examples/92-ui-kit -element scroll
```

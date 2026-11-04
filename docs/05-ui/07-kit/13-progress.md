# Progress

![Progress bars at 0, 30 and 100 percent, one without its value, and one over a range of 10 to 210 at 70](progress.png "ember · progress")

<!-- chrome -->
Matches Chrome within 0.15% of its pixels (threshold 0.20%).
<!-- /chrome -->

```go
loading := kit.Progress(65)
xp := kit.Progress(340, kit.Range(0, 500))
```

| Option | Default | |
|---|---|---|
| `kit.Range(0, 500)` | 0 to 100 | the values at its ends; the percentage counts from the low end |
| `kit.ShowValue(false)` | shown | no percentage over the bar |

```go
kit.SetProgress(xp, 420) // in its own units: 84%, the percentage with it
```

`kit.ProgressBar(loading)` is its `progress` element.

| CSS | |
|---|---|
| `.kit-progress` | the box: 200×20 |
| `progress.kit-progress-bar` | the bar, its fill in `accent-color` |
| `.kit-progress-value` | the percentage, centred over it |

Automation: not recorded; it only shows.

```
go run ./examples/92-ui-kit -element progress
```

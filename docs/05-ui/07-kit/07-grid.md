# Grid

![Grids of cells in dashed frames: two and three columns, gap 10, padding 8, a max-content 1fr grid and two equal columns](grid.png "ember · grid")

<!-- chrome -->
Matches Chrome within 0.28% of its pixels (threshold 0.47%).
<!-- /chrome -->

```go
props := kit.Grid(2, kit.Columns("max-content 1fr"), name, grass, solid, no)
```

| Option | Default | |
|---|---|---|
| `kit.Grid(columns)` | | that many columns, each as wide as its widest cell |
| `kit.Columns(tracks)` | `repeat(n, max-content)` | CSS tracks: `"max-content 1fr"` gives the last column the room left |
| `kit.Gap(px)` | 4 | points between cells |
| `kit.Padding(px)` | 0 | points inside the edge |

| CSS | |
|---|---|
| `.kit-grid` | a CSS grid, gap 4 |

Automation: not seen; with an `id` it prefixes its cells' IDs.

```
go run ./examples/92-ui-kit -element grid
```

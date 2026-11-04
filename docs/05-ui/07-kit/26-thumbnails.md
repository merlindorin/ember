# Thumbnails

![Six 118×112 cells: floor.tsj selected in amber, old.tsj dimmed, a long name cut, one without a picture; then four smaller 84×92 cells, lava selected, the grid focused](thumbnails.png "ember · thumbnails")

<!-- chrome -->
Matches Chrome within 0.17% of its pixels (threshold 0.20%).
<!-- /chrome -->

```go
levels := kit.Thumbnails(kit.Thumbnail("Crypt", "crypt.png"), kit.Thumbnail("Sanctuary", "sanctuary.png"))
```

| Option | Default | |
|---|---|---|
| `kit.ThumbnailSize(w, h)` | 118 × 112 | the size of every cell |
| `kit.Thumbnail(name, "")` | a picture | an empty cell above the name |

| Reading and changing it | |
|---|---|
| `d.Selected(levels)`, `d.SetSelected(levels, i)` | the selected cell, -1 for none |
| `kit.SetDimmed(cell, true)` | greyed out, never selected |

| Event | When |
|---|---|
| `ui.Changed{Node}` | a click or an arrow selects a cell; Node is the Thumbnails |
| `ui.Submitted{Node}` | a double click, Enter or the pad's South on a cell |

| CSS | |
|---|---|
| `.kit-thumbnails` | the well, with `:focus-visible` |
| `.kit-thumbnail` | a cell, with `[aria-selected="true"]` and `[aria-disabled="true"]` |
| `.kit-thumbnail-picture`, `.kit-thumbnail-name` | its picture and its name |

Automation: each cell by the slug of its name (`crypt`); `press`.

```
go run ./examples/92-ui-kit -element thumbnails
```

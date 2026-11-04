# Tooltip

![Tooltips closed, hovered, shown by keyboard focus, on the left, below, on a button, and as a panel of two labels](tooltip.png "ember · tooltip")

<!-- chrome -->
Matches Chrome within 0.25% of its pixels (threshold 0.40%).
<!-- /chrome -->

```go
save := kit.Button("Save", kit.Tooltip(kit.Label("Saves the tileset")))
```

| Option | Default | |
|---|---|---|
| `kit.Left()` on the tip | on the right | the tip on the element's left |
| `kit.Below()` on the tip | on the right | the tip under the element |

The tip becomes the element's child, so it goes on a container or a button, never on an input or an img.

| CSS | |
|---|---|
| `.kit-has-tooltip` | the element holding a tip |
| `.kit-tooltip` | the tip, shown on `.kit-has-tooltip:hover` and `:focus-visible` |
| `.kit-tooltip.kit-left`, `.kit-tooltip.kit-below` | the two other sides |

Automation: the tip is no element of its own and never part of its element's name.

```
go run ./examples/92-ui-kit -element tooltip
```

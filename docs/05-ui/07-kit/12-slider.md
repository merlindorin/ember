# Slider

![Sliders at 0, 30 and 100, hovered, focused, and over a range of 2 to 8 at 5](slider.png "ember · slider")

<!-- chrome -->
Matches Chrome within 0.08% of its pixels (threshold 0.11%).
<!-- /chrome -->

```go
volume := kit.Slider(70, ui.ID("music"))
zoom := kit.Slider(2, kit.Range(1, 4), kit.Step(0.5))
```

| Option | Default | |
|---|---|---|
| `kit.Range(1, 4)` | 0 to 100 | the values at its ends |
| `kit.Step(0.5)` | 1 | what an arrow key adds, where values snap |
| `ui.Disabled()` | enabled | takes no drag, click or key |

```go
v, _ := strconv.ParseFloat(d.Value(volume), 64)
d.SetValue(volume, "40")
```

| Event | When |
|---|---|
| `ui.Edited{Node}` | each value while dragged, clicked or stepped by the arrow keys |
| `ui.Changed{Node}` | a drag released on a new value, a click, an arrow key, automation's set |

| CSS | |
|---|---|
| `.kit-slider` | track, fill (`accent-color`) and thumb, with `:focus-visible` |

Automation: a slider named by its `id`, else its `aria-label`, else a `<label>` round it (a `kit.Field`'s label doesn't name it); `set`.

```
go run ./examples/92-ui-kit -element slider
```

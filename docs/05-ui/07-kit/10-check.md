# Check

![Checks unchecked and checked, in each state: default, hover, focus-visible and disabled, and one on the baseline of a button beside it](check.png "ember · check")

<!-- chrome -->
Matches Chrome within 0.41% of its pixels (threshold 0.52%).
<!-- /chrome -->

```go
grid := kit.Check("Show grid", ui.Checked(true))
```

| Option | Default | |
|---|---|---|
| `ui.Checked(true)` | unchecked | the box filled in the accent colour |
| `ui.Disabled()` | enabled | box and text greyed out, takes no press |

```go
box := kit.CheckBox(grid)
on := d.Checked(box)
d.SetChecked(box, false)
```

| Event | When |
|---|---|
| `ui.Edited{Node}`, `ui.Changed{Node}` | toggled: a click on the box or its text, Enter or Space while focused, the pad's South, or automation; `Node` is `kit.CheckBox(grid)` |

| CSS | |
|---|---|
| `.kit-check` | the label round box and text, with `:hover` and `.kit-disabled` |
| `.kit-check-box` | the box, with `:checked` and `:focus-visible` |
| `.kit-check-text` | the text |

Automation: a toggle named by its text (`show-grid`); `press`, `set`.

```
go run ./examples/92-ui-kit -element check
```

# Number field

![Number fields stepping by 1 and by 0.5, and one empty with a placeholder, in each state: default, hover, focus-visible, read-only](number-field.png "ember · number field")

<!-- chrome -->
Matches Chrome within 0.31% of its pixels (threshold 0.40%).
<!-- /chrome -->

```go
width := kit.NumberField(32, kit.Range(1, 256), ui.ID("width"))
zoom := kit.NumberField(1, kit.Range(0.25, 4), kit.Step(0.25))
```

| Option | Default | |
|---|---|---|
| `kit.Range(1, 256)` | unbounded | typed values kept within it |
| `kit.Step(0.25)` | 1 | what Up and Down add (ten with Shift, or the wheel while focused), where values snap |
| `ui.Placeholder("Height")` | none | muted text while empty |
| `kit.ReadOnly()` | editable | selected and copied, not edited or stepped |

```go
v, _ := strconv.ParseFloat(d.Value(width), 64)
d.SetValue(width, "48")
```

| Event | When |
|---|---|
| `ui.Edited{Node}` | every keystroke, every step by Up, Down or the wheel |
| `ui.Changed{Node}` | Enter, or the focus leaving, after a change: the text taken within the range on a step, or the last value back when it isn't a number |
| `ui.Submitted{Node}` | Enter |

| CSS | |
|---|---|
| `.kit-number-field` | its look, with `::placeholder`, `:focus` and `:focus-visible` |

Automation: its `id`, else its `aria-label`, else a `<label>` round it, else its placeholder (`width`); `type`, `set`, `select-all`, `submit`.

```
go run ./examples/92-ui-kit -element number-field
```

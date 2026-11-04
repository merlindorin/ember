# Choice

![Choices with an item selected, with none, segmented, hovered, focused, and as colour swatches](choice.png "ember · choice")

<!-- chrome -->
Matches Chrome within 0.33% of its pixels (threshold 0.50%).
<!-- /chrome -->

```go
layer := kit.Choice(0, []*ui.Node{kit.Label("Tiles"), kit.Label("Objects"), kit.Label("Lights")})
```

| Option | Default | |
|---|---|---|
| `selected` | | the item selected, ringed in the accent colour; `-1` for none |
| `kit.Segmented()` | separate cells | the items joined as tabs |
| `kit.Fill()` | as wide as its items, at the start of a form's column | the items share its width equally, words kept whole: in a form, it ends flush with the fields above |

```go
i := kit.Chosen(layer)
d.SetChecked(layer.Children()[2], true)
```

| Event | When |
|---|---|
| `ui.Edited{Node}`, `ui.Changed{Node}` | another item selected: a click, Enter, Space or the pad's South on it, the arrows, or automation; `Node` is the item's button |
| `ui.Clicked{Node}` | every press of an item, selected already or not |

| CSS | |
|---|---|
| `.kit-choice` | the row of items |
| `.kit-choice-item` | each item, with `:hover`, `:active`, `:focus-visible` and `[aria-checked="true"]` |
| `.kit-choice.kit-segmented` | items joined |

Automation: each item is a select named by its text (`tiles`), else its button's `aria-label`; `press`.

```
go run ./examples/92-ui-kit -element choice
```

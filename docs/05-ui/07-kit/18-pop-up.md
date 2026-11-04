# Pop-up

![Pop-ups showing an item, a placeholder, hovered, pressed and focused from the keys](pop-up.png "ember · pop-up")

<!-- chrome 18-popup -->
Matches Chrome within 0.33% of its pixels (threshold 0.44%).
<!-- /chrome -->

```go
mode := kit.PopUp(0, []string{"Windowed", "Borderless", "Fullscreen"})
shape := kit.PopUp(-1, shapes, ui.Placeholder("Choose…"))
```

| Option | Default | |
|---|---|---|
| `ui.Placeholder("Choose…")` | none | the dimmed text while no item is chosen |

```go
i := d.Selected(mode)  // -1 for none
d.SetSelected(mode, 2) // no event
```

| Event | When |
|---|---|
| `ui.Edited{Node}`, then `ui.Changed{Node}` | an item chosen from its list, opened by a click, Enter, Space or the pad's South; or a press on it, released over an item |

| CSS | |
|---|---|
| `select.kit-pop-up` | its look and chevron, with `:hover`, `:active` and `:focus-visible` |

Automation: its `id`, else `data-name`, `aria-label`, its label's text (`kit.Field("Window mode", …)` → `window-mode`), then its placeholder, never its items; with none of them it isn't recorded; `press`, `select`.

```
go run ./examples/92-ui-kit -element pop-up
```

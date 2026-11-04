# Panel

![Panels plain, titled, checked, glowing black and in the accent, a collapsible panel open and folded, and cards at rest, hovered and focused](panel.png "ember · panel")

<!-- chrome -->
Matches Chrome within 0.27% of its pixels (threshold 0.34%).
<!-- /chrome -->

```go
settings := kit.Panel(kit.Title("Settings"), content)
lights := kit.CollapsiblePanel("Lights", true, content)
newGame := kit.Card(kit.Label("New game"))
```

| Option | Default | |
|---|---|---|
| `kit.Title(text)` | none | a title bar, first |
| `kit.Checked()` | | the toggled look: accent border |
| `ui.Class("kit-glow")` | | a dark halo |
| `kit.Glow(c, size)` | | a halo of colour `c`, fading over `size` points |
| `kit.CollapsiblePanel(title, open)` | | the title bar folds and opens it |
| `kit.Card(args…)` | | a pressable panel on the window colour, with a glow |

Reading and changing it:

```go
settings.AddClass("checked")
doc.SetOpen(lights, false) // folds it, without events; SetAttr("open") is overwritten
open := doc.Open(lights)
```

| Event | When |
|---|---|
| `ui.Toggled{Node}` | a collapsible panel folded or opened: a click on its bar, Enter or Space |
| `ui.Clicked{Node}` | a card pressed: a click, Enter, Space, the pad's South or automation's press; also a collapsible panel's bar |

| CSS | |
|---|---|
| `.kit-panel` | panel colour, border, 6-point corners |
| `.kit-panel-title` | the title bar, accent text |
| `.kit-panel.kit-checked`, `.kit-panel.kit-glow` | the two looks |
| `.kit-panel.kit-card` | the card, with `:focus-visible` |

Automation: a collapsible panel's bar by the slug of its title (`lights`), a card by its text (`new-game`); `press`.

```
go run ./examples/92-ui-kit -element panel
```

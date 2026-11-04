# Automation

```go
auto := automation.New(automation.Recording())
ember.New(ember.Automation(auto)).AddEntities(win)
```

| An element's ID | |
|---|---|
| its `id` | `ui.ID("save")` → `save` |
| else its `data-name` | `ui.Attr("data-name", "slot-1")` → `slot-1` |
| else its `aria-label` | `ui.Attr("aria-label", "Close")` → `close` |
| else the slug of its text, label or placeholder | `kit.Button("Save tileset")` → `save-tileset` |
| prefixed by its ancestors' ids and data-names | `settings/save` |
| text a `data-name` names | `ui.Span(ui.Attr("data-name", "stock"), ui.Text("12"))` → a label `stock` reading `12`; a named box of elements only scopes |
| none of these: not recorded | a pop-up with neither id nor label |

A label's text never includes the control it labels.

An element that draws its own value, such as a `ui.Canvas`, tells automation what it shows with WAI-ARIA:

```go
orb := ui.Canvas(draw, ui.ID("health"), ui.Attr("role", "progressbar"), ui.Attr("aria-valuenow", "75"))
```

| `role` with `aria-valuenow` | Recorded as |
|---|---|
| `progressbar` | a bar, its value the share of `aria-valuemin` to `aria-valuemax` (0 to 100 by default): `0.75` |
| `meter` | a meter, the value as is |
| `slider` | a slider, the value as is |

| Action | On |
|---|---|
| `press` | buttons, checks, summaries, menu items |
| `set` | a control's value |
| `type` | text at a text field's caret |
| `select` | an option, a tab, a row: selects only; `press` on a row or option selects and activates it |
| `context` | an element with `OnContextMenu`: a right click |
| `drag` | a `Draggable` element, its value the target's path: dragged onto it |

Served over Connect when `EMBER_AUTOMATION=host:port` is set, as for every ember app.

```
go run ./examples/92-ui-kit -element settings
```

# Alert

![An alert over a dimmed page: an amber exclamation mark, Save the tileset?, a message, Don't save in red on the left, Cancel and a focused Save on the right](alert.png "ember · alert")

<!-- chrome -->
Matches Chrome within 0.15% of its pixels (threshold 0.20%).
<!-- /chrome -->

```go
ask := kit.Alert("Save the tileset?", "Your changes to walls.tsj will be lost if you close it without saving.",
	kit.Button("Don't save", kit.Destructive()),
	kit.Button("Cancel"),
	kit.Button("Save", kit.Primary()))
d.Body().Append(ask)
d.ShowModal(ask)
```

| Option | Default | |
|---|---|---|
| `kit.Primary()` on a button | its first button that isn't destructive | focused first, pressed by Enter |
| `kit.Destructive()` on a button | right | stands on the left |
| `ui.Value("discard")` on a button | its text | what it answers |

| Event | When |
|---|---|
| `ui.Closed{Node, Value}` | a button answers it (its value), or Escape or the pad's East (`""`) |

| CSS | |
|---|---|
| `.kit-alert` | the box, hidden without `[open]` |
| `.kit-alert-icon`, `.kit-alert-title`, `.kit-alert-message` | its parts |
| `.kit-alert-buttons`, `.kit-alert-buttons-left`, `.kit-alert-buttons-right` | the buttons' rows |

Automation: the alert by the slug of its title (`save-the-tileset`), its buttons by the slug of their text (`don-t-save`, `save`); `press`. `kit.SetAlert(alert, title, message)` asks it again about something else.

```
go run ./examples/92-ui-kit -element alert
```

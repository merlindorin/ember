# Label

![Labels default, wrapped, truncated, centred and right-aligned in dashed boxes, then the caps, form-label, alert-title and alert-message styles](label.png "ember · label")

<!-- chrome -->
Matches Chrome within 0.42% of its pixels (threshold 0.60%).
<!-- /chrome -->

```go
hp := kit.Label("Health: 100")
bio := kit.Label(text, kit.Wrap())
```

| Option | Default | |
|---|---|---|
| `kit.Wrap()` | one line | wraps in its box, breaking long words |
| `kit.Truncate()` | one line | cut at its box with an ellipsis |
| `kit.Center()`, `kit.Right()` | left | aligned in its box |
| `kit.Caps()` | body text | small, muted, upper case |
| `kit.FormLabel()` | body text | small and muted: a field's name |
| `kit.AlertTitle()`, `kit.AlertMessage()` | body text | an alert's bold title and muted message |

Reading and changing it:

```go
hp.SetText("Health: 80")
hp.Text() // "Health: 80"
```

| CSS | |
|---|---|
| `.kit-label` | one line (`white-space: nowrap`) |
| `.kit-label.kit-wrap`, `.kit-label.kit-truncate`, `.kit-label.kit-center`, `.kit-label.kit-right` | the four layouts, each a block |
| `.kit-caps`, `.kit-form-label`, `.kit-alert-title`, `.kit-alert-message` | the text styles |

Automation: only with an `id` (`ui.ID("hp")`), a label reading its text; no action.

```
go run ./examples/92-ui-kit -element label
```

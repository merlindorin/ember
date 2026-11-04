# Text area

![Text areas three rows high with wrapped text and with a placeholder, in each state: default, hover, focus-visible, read-only; then one scrolling down and one NoWrap scrolling sideways](text-area.png "ember · text area")

<!-- chrome 15-textarea -->
Matches Chrome within 0.89% of its pixels (threshold 1.20%).
<!-- /chrome -->

```go
notes := kit.TextArea("", ui.ID("quest-notes"), ui.Placeholder("Quest notes"), kit.Rows(6))
```

| Option | Default | |
|---|---|---|
| `ui.Placeholder("Quest notes")` | none | muted text while empty |
| `kit.Rows(6)` | 5 | lines high; more scroll, with a bar |
| `kit.NoWrap()` | wraps | lines kept whole, scrolling sideways |
| `kit.MaxLength(500)` | no limit | typing stops at 500 characters |
| `kit.ReadOnly()` | editable | selected and copied, not edited |

It scrolls its own lines to keep the caret in view as the player types or moves it, and a click lands on the line under the pointer.

```go
s := d.Value(notes)
d.SetValue(notes, "Find the crypt key.")
```

| Event | When |
|---|---|
| `ui.Edited{Node}` | every keystroke, Enter's new line, paste, cut or undo |
| `ui.Changed{Node}` | the focus leaving after an edit |
| `ui.Submitted{Node}` | Command+Enter on macOS, Control+Enter elsewhere |

| CSS | |
|---|---|
| `.kit-text-area` | its look, with `::placeholder`, `:focus` and `:focus-visible` |
| `.kit-text-area.kit-no-wrap` | `NoWrap` |

Automation: its `id`, else its `aria-label`, else a `<label>` round it, else its placeholder, never its text (`quest-notes`); `type`, `set`, `select-all`.

```
go run ./examples/92-ui-kit -element text-area
```

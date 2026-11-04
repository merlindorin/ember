# button

![Two buttons in the default look, Save and a disabled Delete](button.png "ember · button")

```go
save := ui.Button(ui.ID("save"), ui.Text("Save"))
ui.Button(ui.Text("Delete"), ui.Disabled())
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its label; children may be any inline elements |
| `ui.Disabled()` | takes no press and no focus; `:disabled` |
| `ui.TabIndex(i)` | its place in the Tab order; -1 leaves it out |

| Default CSS | |
|---|---|
| `button` | `display: inline-block; padding: 1px 6px; border: 1px solid #767676; border-radius: 3px; background-color: #efefef; text-align: center`, font and colour inherited |
| `button:hover`, `button:active` | `#e5e5e5`, `#f5f5f5` |
| `button:disabled` | text, background and border at 30% |

| Event | When |
|---|---|
| `ui.Clicked{Node}` | a click, Enter or Space while focused, the pad's South, or automation's press |

Automation: its `id`, else `data-name`, `aria-label`, then the slug of its text (`save`); `press`.

```
go run ./examples/93-html-elements -element button
```

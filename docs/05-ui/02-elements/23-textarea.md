# textarea

![A quest log's notes in a textarea of four rows, wrapped](textarea.png "ember · textarea")

```go
ui.Textarea(ui.ID("notes"), ui.Attr("rows", "4"), ui.Attr("cols", "30"),
	ui.Text("Find the old well north of the village."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its starting text; `doc.Value(n)` reads it now, `doc.SetValue(n, v)` sets it |
| `ui.Placeholder(s)` | its text while empty |
| `rows`, `cols` | its size in lines (2) and average characters (20) |
| `readonly`, `maxlength` | kept as is, or held to that many characters |
| `ui.Disabled()` | takes no edit and no focus; `:disabled` |

| Default CSS | |
|---|---|
| `button, input, select, textarea` | `appearance: none; margin: 0; padding: 0; border: 0; background: none`, font, colour, line height and text alignment inherited; `display: inline-block` |
| `input, select, textarea` | `padding: 1px 2px; border: 1px solid #767676; border-radius: 2px; background-color: #fff` |
| `textarea` | `white-space: pre-wrap; overflow: auto` |
| `::placeholder` | `color: #757575` |
| `input:disabled, textarea:disabled` | text `#545454`, background and border at 30% |

| Event | When |
|---|---|
| `ui.Edited{Node}` | every keystroke |
| `ui.Changed{Node}` | the focus leaves after an edit |
| `ui.Submitted{Node}` | the submit shortcut: Command+Enter on macOS, Control+Enter elsewhere |

Automation: its `id`, else `data-name`, `aria-label`, its label's text, then its placeholder (`notes`); `type`, `set`, `select-all`.

```
go run ./examples/93-html-elements -element textarea
```

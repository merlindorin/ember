# select

![A difficulty select showing Normal, and a class select showing its placeholder, Pick one](select.png "ember · select")

```go
ui.Select(ui.Placeholder("Pick one"),
	ui.Option(ui.Value("warrior"), ui.Text("Warrior")),
	ui.Option(ui.Value("mage"), ui.Text("Mage")),
)
```

| Attribute | |
|---|---|
| its [`option`](22-option.md) children | what it offers; the chosen one's text shows; `doc.Selected(n)`, `doc.SetSelected(n, i)`, `doc.Value(n)` |
| `ui.Placeholder(s)` | a ui extension: it starts with nothing chosen and shows `s`, dimmed |
| `ui.Disabled()` | takes no press and no focus; `:disabled` |

| Default CSS | |
|---|---|
| `button, input, select, textarea` | `appearance: none; margin: 0; padding: 0; border: 0; background: none`, font, colour, line height and text alignment inherited; `display: inline-block` |
| `input, select, textarea` | `padding: 1px 2px; border: 1px solid #767676; border-radius: 2px; background-color: #fff` |
| `input, select` | `white-space: nowrap; overflow: hidden` |
| `select` | `padding-right: 20px; background-image: url(ui:select-arrow); background-repeat: no-repeat; background-position: right 6px center` |
| `select:disabled` | grey text, border at 30%, `opacity: 0.7` |

Its width is its widest option's, its placeholder counted while it shows. Open, its highlighted option takes its `accent-color`.

| Event | When |
|---|---|
| `ui.Edited{Node}`, `ui.Changed{Node}` | another option chosen: a click opens the list and a click chooses, or a press opens it and releasing over an option chooses |

Automation: its `id`, else `data-name`, `aria-label`, its label's text, then its placeholder (`difficulty`); `press` opens it, `select` chooses an option by index, value or text.

```
go run ./examples/93-html-elements -element select
```

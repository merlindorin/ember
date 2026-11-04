# option

![A difficulty select showing Normal, its chosen option](option.png "ember · option")

```go
ui.Select(
	ui.Option(ui.Value("easy"), ui.Text("Easy")),
	ui.Option(ui.Value("normal"), ui.Text("Normal"), ui.Selected()),
	ui.Option(ui.Value("nightmare"), ui.Text("Nightmare"), ui.Disabled()),
)
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text, in the list and in the select once chosen |
| `ui.Value(v)` | the select's value when chosen; its text without one |
| `ui.Selected()` | the select starts with it chosen |
| `ui.Disabled()` | shown dimmed, never chosen |

| Default CSS | |
|---|---|
| `option` | `display: block; padding: 0 2px 1px`, in its select's open list |
| `option:disabled` | `color: gray` |

Automation: through its select's `select`.

```
go run ./examples/93-html-elements -element option
```

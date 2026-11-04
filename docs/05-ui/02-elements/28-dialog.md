# dialog

![A quit dialog: Quit to the title screen? over Stay and Quit buttons, in a black frame](dialog.png "ember · dialog")

```go
quit := ui.Dialog(ui.ID("quit"),
	ui.P(ui.Text("Quit to the title screen?")),
	ui.Button(ui.Text("Stay"), ui.Attr("formmethod", "dialog"), ui.Value("stay")),
	ui.Button(ui.Text("Quit"), ui.Attr("formmethod", "dialog"), ui.Value("quit")),
)
doc.Show(quit)      // the page stays live round it
doc.ShowModal(quit) // above a backdrop; the page takes neither pointer nor focus
doc.Close(quit, "stay")
doc.Open(quit)
```

| Attribute | |
|---|---|
| `open` | set while it shows, by `Show` and `ShowModal`, removed by `Close` |
| a button's `formmethod="dialog"` and `value` | the button closes it, its value the return value |
| `autofocus` on a child | where the focus goes when it opens; a modal's first control otherwise |

| Default CSS | |
|---|---|
| `dialog` | `display: block; position: absolute; left: 0; right: 0; margin: auto; width: fit-content; height: fit-content; padding: 1em; border: solid; background-color: #fff; color: #000`: in the flow where it stands, centred across |
| `dialog:modal` | `position: fixed; overflow: auto; top: 0; bottom: 0`, at most the window less `6px + 2em`: centred in the window |
| `dialog::backdrop` | `background-color: rgba(0, 0, 0, 0.1)`; set it to recolour or drop the dimming |
| `dialog:not([open])` | `display: none` |

| Event | When |
|---|---|
| `ui.Closed{Node, Value}` | closed by `Close` or a `formmethod="dialog"` button, or by Escape or the pad's East while modal (`Value` empty) |

Automation: while open, its `id` (`quit`), else `data-name`, `aria-label`, then the slug of its text; labelled by its `aria-label` or first heading, its value whether it is modal.

```
go run ./examples/93-html-elements -element dialog
```

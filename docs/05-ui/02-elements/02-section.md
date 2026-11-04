# section

![A settings screen in two dashed sections, Audio and Video](section.png "ember · section")

```go
ui.Section(ui.ID("audio"),
	ui.H3(ui.Text("Audio")),
	ui.P(ui.Text("Music 80%, effects 60%")),
)
```

| Default CSS | |
|---|---|
| `section` | `display: block` |

Laid out as a div: it says the content is one themed part of a screen, usually with its own heading.

Automation: not recorded, unless it has an `id` (then a label of its text), as every block; no landmark role yet; its `id` or `data-name` prefixes its descendants' IDs. With `ui.ID("audio")`, a control inside is `audio/…`.

```
go run ./examples/93-html-elements -element section
```

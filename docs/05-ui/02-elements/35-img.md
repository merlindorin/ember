# img

![A framed panel picture at its own 24×24 size, then three times larger with sharp pixels](img.png "ember · img")

```go
doc := ui.NewDocument(ui.Images(fstest.MapFS{"panel.png": {Data: panelPNG}}))

ui.Img(ui.Attr("src", "panel.png"), ui.Attr("alt", "A panel"))
ui.Img(ui.Attr("src", "panel.png"), ui.Style("width: 72px; image-rendering: pixelated"))
```

| Attribute | |
|---|---|
| `src` | a PNG in the Document's `ui.Images` and stylesheets' file systems, the last added first; one missing is in `doc.Warnings()` |
| `alt` | its name for automation (an `aria-label` wins); a picture that isn't found draws Chrome's broken-image icon and the alt text, wrapped and clipped to the box, a silver frame round the icon without `alt`, nothing with `alt=""`; unsized, it's as wide as the icon and its text |

Its natural size is one point per pixel; with one side set, the other keeps the ratio. `object-fit`, `object-position` and `image-rendering: pixelated` apply.

| Default CSS | |
|---|---|
| `img` | `display: inline-block` (Chrome: inline; ui has no `vertical-align` for inline pictures) |

Automation: named by its `alt`, recorded as a label; not recorded with neither `alt` nor `id`.

```
go run ./examples/93-html-elements -element img
```

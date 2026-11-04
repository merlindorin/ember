# Image

![Colour rects, nine-patches stretched into a dialog, a button and a bar, then a 24×24 picture as it is, scaled, kept, kept centred, fitted, fitted centred, covering, and tiled](image.png "ember · image")

<!-- chrome -->
Matches Chrome within 0.15% of its pixels (threshold 0.18%).
<!-- /chrome -->

```go
portrait := kit.Image("panel.png", kit.KeepAspectCentered())
dialog := kit.NinePatch("panel.png", 8, 8, 8, 8, kit.Padding(4), title, message)
floor := kit.TileImage("panel.png", ui.Class("tiles"))
swatch := kit.ColorRect("var(--accent-color)", ui.Class("swatch"))
```

`src` is a file of the Document's stylesheets (`ui.Stylesheet(fsys, …)`).

| Option | Default | |
|---|---|---|
| `kit.Image(src)` | the picture's size | stretched to fill its box |
| `kit.Keep()`, `kit.KeepCentered()` | | its own size, at the top left or the centre |
| `kit.KeepAspect()`, `kit.KeepAspectCentered()` | | fitted inside, its shape kept |
| `kit.KeepAspectCovered()` | | covers the box, cut to it |
| `kit.NinePatch(src, left, top, right, bottom)` | | corners and edges kept at those margins (image pixels), the rest stretched; children inside |
| `kit.TileImage(src)` | no size | repeated from the top left |
| `kit.ColorRect(c)` | no size | filled with a CSS colour |

Changing it:

```go
portrait.SetAttr("src", "heron.png")
```

| CSS | |
|---|---|
| `.kit-image`, `.kit-image.kit-keep`… | `object-fit` and `object-position` of each mode |
| `.kit-nine-patch` | a block; the patch is its `border-image` |
| `.kit-tile-image` | `background-repeat: repeat` |
| `.kit-color-rect` | its fill |

Automation: only with an `id`, a label; no action.

```
go run ./examples/92-ui-kit -element image
```

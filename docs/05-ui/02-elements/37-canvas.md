# canvas

![A canvas of three rooms on a dark map, an orange marker where the pointer is](canvas.png "ember · canvas")

```go
minimap = ui.Canvas(func(c *twod.Canvas) {
	c.Rect(geom.Rect{W: c.Size().X, H: c.Size().Y}, dark)
	if at, ok := doc.PointerOn(minimap); ok {
		c.Circle(at, 5, orange)
	}
}, ui.Attr("width", "200"), ui.Attr("height", "150"))
doc.Invalidate(minimap) // when what it draws changes
```

| Attribute | |
|---|---|
| `width`, `height` | its natural size in points, 300 × 150 without them; CSS sizes it as an img, one side keeping the ratio |

| | |
|---|---|
| the drawing | in its content box: the origin its top-left, `c.Size()` its size in points, clipped |
| when draw runs | first shown, after `doc.Invalidate(n)`, and when its box moves or resizes; otherwise its last drawing is replayed, so a redraw costs only its own draw |
| `doc.PointerOn(n)` | the pointer in its content box, in points; false off it |
| `c.Image(r, tex, uv, tint)` | a texture over `r`, a render texture included: what a camera draws `twod.Into` it |
| not yet | an ancestor's `opacity` below 1 doesn't fade it |

A minimap: a camera draws the world into a render texture, and the canvas shows it.

```go
tex := twod.NewRenderTexture(256, 256)
cam := twod.NewCamera(twod.Into(tex), twod.Follow(hero), twod.Zoom(0.25))
minimap := ui.Canvas(func(c *twod.Canvas) {
	c.Image(geom.Rect{W: c.Size().X, H: c.Size().Y}, tex, geom.Rect{W: 1, H: 1}, color.White)
}, ui.Attr("width", "160"), ui.Attr("height", "160"))
```

| Default CSS | |
|---|---|
| `canvas` | `display: inline-block` (Chrome: inline; ui has no `vertical-align` for inline pictures) |

```
go run ./examples/93-html-elements -element canvas
```

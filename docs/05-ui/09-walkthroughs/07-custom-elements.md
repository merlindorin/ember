# Custom elements

A game can make its own UI elements and draw them itself. Here a red health orb sits next to built-in ones: a label and two buttons, Hurt and Heal.

![A health orb beside its label and two buttons](orb.png "ember · custom elements")

## The element

`Orb` holds a `ui.Canvas`, so the layout places it like any element and the page's sheet sizes it. Its draw function paints on a `twod.Canvas`, the same canvas 2D games draw on, in the element's own space: `c.Size()` is the size the layout gave it. A canvas keeps its last drawing, so `SetHealth` asks for a new one with `doc.Invalidate`. Its `role="meter"` and `aria-valuenow`, kept current, tell automation what it shows ([Automation](../06-automation.md)).

[orb.go](../../../examples/17-custom-elements/orb.go)

## Drawing on the canvas

`c.RoundRectCorners(rect, radii, colour)` fills a box whose corners each take their own radius, top-left, top-right, bottom-right then bottom-left as in CSS, scaled down together when two on one side would overlap: `[4]float32{8, 8, 0, 0}` is a tab, one sharp corner a speech bubble; `c.BorderCorners(rect, radii, thickness, colour)` strokes the same outline inside the box. `c.Glow(rect, radius, spread, colour)` draws a soft halo around a rectangle with corners rounded by radius, fading out over spread points, as a kit panel's glow does. `c.Triangles(points, indices, colour)` fills any shape cut into triangles, and `c.TrianglesColored(points, colours, indices)` gives each point its own colour, blended across, so a shape can fade at its rim: a light disc on a `Blend(Multiply)` layer, bright in the middle and gone at its edge, or a line-of-sight polygon.

```go
points, colours := []geom.Vec2{{}}, []color.Color{color.White}
var indices []int
for i := range 32 {
	points = append(points, geom.FromAngle(float32(i)*2*geom.Pi/32).Scale(radius))
	colours = append(colours, color.White.Alpha(0))
	indices = append(indices, 0, i+1, (i+1)%32+1)
}
c.TrianglesColored(points, colours, indices)
```

## Changing it

`Health`'s `Hurt` and `Heal` are the buttons' `ui.OnClick` callbacks: they set the orb and the label.

[health.go](../../../examples/17-custom-elements/health.go)

## With built-in elements

The orb goes in a row with a column of kit elements, inside a panel, and they lay out together.

[main.go](../../../examples/17-custom-elements/main.go)

## Try it

[17-custom-elements](../../../examples/17-custom-elements)

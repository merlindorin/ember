# 2D

2D games are made of entities too: sprites, text, lines, shapes, tile maps and cameras from `twod`. Each is made with options, added as a window's child (one that belongs to no window shows in the primary one), and changed while the game runs with setters.

## Placing things

With no camera, one unit is one point and the origin is the window's top-left corner, y growing down. A camera with a `Resolution` makes one unit one pixel of that resolution instead, as in [Pixel-perfect](13-pixel-perfect.md). `At` places an entity's pivot: the centre by default for anything with a size, such as a sprite, a text or a nine-patch, where `Pivot(0, 0)` makes it the top-left corner and `Pivot(0.5, 1)` the feet; the top-left corner of its first cell for a tile map; and the origin of its points for a line or a polygon. `Size(w, h)` fits a sprite or a nine-patch to a box. `Scale`, `Rotation` and `Tint` work on any of them. `Color` is the colour a text, a line or a polygon is drawn in, while `Tint` multiplies whatever an entity draws, its children included, for flashes, fades and team colours; a white text tinted red shows red. A child is placed from its parent: `Add` makes children that move, turn and leave with it, and returns the parent, so a window or a layer is built in one call.

## Draw order

Layers draw in number order: `twod.NewLayer(1)` is in front of layer 0. Within a layer, `Z` orders entities, and otherwise they draw in the order they were added; `SortByY` draws them by their y instead, for top-down scenes. 2D works alone or with a 3D scene in the same window: layers draw over the 3D scene, and negative layers behind it, for a backdrop.

## The pages

They go from drawing one thing to whole levels: sprites and shapes first, then layers and cameras, then levels made of tiles that the hero collides with, then the effects a layer or a camera can apply, how things move, lights, pictures the game paints a piece at a time, and last variants of a picture, recoloured or masked, made at load.

Text is a 2D entity too, but fonts are a subject of their own: they have their own section, [Text](../07-text/README.md).

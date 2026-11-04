# Tiled pictures

A picture as large as the world, endless if need be, that the game paints a piece at a time: fog of war, a map of what's been explored, the light of each cell. Here the garden is hidden by fog that lifts as the hero walks, and an ambient map lights a clearing at its heart, darkening toward its edges.

![The hero walking through the garden at night, the fog of war lifting cell by cell round him](tiled-pictures.gif "ember · tiled pictures")

## The garden

The garden of [Point lights](19-lights/02-point-lights.md), with the hero and his torch. The fog is on a layer of its own above the world, not lit, so it covers everything under it and no light brightens it; `light.Unlit()` on the picture in a lit layer would do too. The camera takes its ambient light from the second picture.

[main.go](../../examples/42-tiled-pictures/main.go)

## Fog of war

`twod.NewTiledPicture(paint, opts…)` is a 2D entity: in a layer, it draws the tiles of itself its camera shows. Each tile is painted by the game the first time it's in view, through `paint`, which gets a `twod.PictureTile`: `Image`, an `*image.RGBA` to write into in place, `Area`, the texels to paint in it, a one-texel border included, `Cell(x, y)`, which texel of the whole picture a pixel is, `World`, the tile's rectangle in the picture's space, and `X`, `Y`, which tile it is. Paint only inside `Area`.

`twod.TexelSize(w, h)` is the points a texel covers, 1×1 by default; here 32×32, so a texel is a cell of the map. The fog paints each cell dark, or transparent once seen. As the hero walks, `Reveal` marks the cells round him seen and calls `MarkDirty(rect)`, which paints again just the texels under rect, in the picture's space, when they're next drawn; `Forget()` paints everything again.

[fog.go](../../examples/42-tiled-pictures/fog.go)

## An ambient map

`light.AmbientMap(pic)` is a camera option: each spot's ambient light is its texel in `pic`, painted as a colour in sRGB bytes, so `color.Gray(0.5).SRGB()` is half light. The picture is in the world's space and needn't be in any layer. Where it paints nothing, transparent, the camera's `Ambient` shows. A tiled picture blends between texels by default, for light and gradients; `twod.Pixelated()` keeps the fog's cells sharp. Lights, the sun, glows and probes all work on top of it. `light.SetAmbientMap(camera, pic)` changes it while the game runs, nil for none, and `light.AmbientMapOf(camera)` reads it; to light a room as it opens, paint its cells again with `MarkDirty`.

[shade.go](../../examples/42-tiled-pictures/shade.go)

Unchanged from [36-point-lights](../../examples/36-point-lights): garden.go, hero.go, point.go, walk.go.

## What it costs

Tiles are 256 texels a side by default, `twod.TileTexels(n)`, and live in atlas pages of 16 MB each, on the CPU and the GPU, 49 tiles a page; a view's tiles draw as one batch per page. Tiles in view are always painted before the frame draws, so nothing shows half done, and `twod.PaintAhead(n)` paints the tiles round the view a frame early, 4 by default. Out of view, `twod.KeepTiles(n)` tiles stay painted, 64 by default; past that the least recently seen give their place up and are painted again if they come back. Only painted rectangles go to the GPU. Pages are only ever added, and each one past the second logs a line, so a view that keeps growing shows; `Pages()` and `Tiles()` count them.

## Try it

[42-tiled-pictures](../../examples/42-tiled-pictures)

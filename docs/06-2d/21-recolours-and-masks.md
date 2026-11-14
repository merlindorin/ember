# Recolours and masks

Variants of a picture, made once when the game loads them: the same crate in other colours, the same wall cut to a shape. A variant is a new texture, drawn like any other, so it costs nothing per frame ([decision 0069](../99-decisions/0069-variants-of-a-picture-are-made-at-load.md)). Above, the crate in its own colours, through a palette made in code and through one loaded from a picture; below, a wall tile as it is, cut by a disc and by the same disc inverted.

![Above, three crates: wood, moss and stone; below, a brick wall tile, the same tile cut to a disc, and the tile with a disc-shaped hole](recolours-and-masks.png "ember · recolours and masks")

## Recolours

A `twod.Palette` swaps exact colours: `twod.NewPalette(twod.Swap(from, to)…)` makes one in code, and its `Swaps()` lists them. `twod.Recolour(tex, pal)` gives a new texture where each colour the palette swaps becomes its `to`, everywhere in the picture, each frame and tile of a sheet included. Colours match exactly, as 8-bit sRGB; alpha is kept, and every colour the palette doesn't name stays as drawn. A palette that matches no colour of the picture is an error, so a palette written for another picture doesn't pass unnoticed.

`twod.LoadPalette(fsys, name)` reads a palette from a picture of two rows of square swatches: the colours to swap above, what they become below. `stone.png` holds the crate's four browns over four greys. `twod.Recoloured(pal)`, given to `LoadTexture`, recolours the picture as it loads, and the texture's `Recoloured()` says which palette it went through.

## Masks

`twod.Masked(tex, region, mask, maskRegion)` keeps the part of a texture where a mask is opaque: a new texture of `region`'s size, whose alpha is the picture's times the mask's, so a mask's clear pixels cut, its opaque ones keep, and those between fade. Only the mask's opacity counts, never its colour. `twod.Inverted()` keeps where the mask is clear instead. Both regions are whole pixels of one size, inside their textures; one that crosses its texture's edge, or two of different sizes, is an error. Here `disc.png`, an opaque disc on clear, cuts the brick tile of `tiles.png`.

A mask that moves with the game, a light's beam or a window over the world, belongs on a layer instead: see [Blend modes and masks](14-blend-modes.md).

## Made once

A recolour or a mask runs once, on the CPU, and is cached by its source and its operation: asking twice for the same crate through an equal palette gives the same texture, so a hundred green crates share one picture. They compose in one order: the recolour, then the mask, then the draw's `Tint`.

[main.go](../../examples/43-recolours-and-masks/main.go)

## In Tiled

A level sets its variants in Tiled, as properties the loader reads; the loader is given the palettes they name and the tileset of the masks:

```go
masks, err := twod.LoadTiledTileset(files, "masks.tsj")
if err != nil {
	log.Fatal(err)
}
level, err := twod.LoadTiled(files, "level.tmj", twod.Palettes(map[string]*twod.Palette{"dark": dark, "moss": moss}), twod.Masks(masks))
```

| Property | Kind | On | Meaning |
|---|---|---|---|
| `tint` | color | a tile, a tile object, a layer | multiplies its colours and opacity |
| `recolour` | string | a tile, a tile object, a layer | the palette, by its name in `twod.Palettes`, its colours are swapped by |
| `mask` | string | a tile, a tile object | the mask, by its tile's `name` in the `twod.Masks` tileset, that cuts it |

![Left, the fixture's two tiles, water with a deep corner and stone, and its two masks, half, keeping the right half, and hole, clear in the middle with a half-alpha rim; right, its map as drawn: the water darkened by its layer's recolour, the stone left as it is and tinted red by its tile, the top-left water cut to its right half and the one below to its left half, flipped, the stone beside them holed with a soft rim; below, the pool object recoloured black but for its corner, and the pillar holed and tinted again](recolours-and-masks-tiled.png "ember · recolours and masks in Tiled")

A mask layer cuts a whole region: a tile layer of class `mask`, whose `cuts` property names the tile layer it cuts, a sibling, else the map's one layer of that name. Each of its cells holds a tile of the mask tileset, found by the tile's `name`, and cuts the cell under it; a cell flipped or rotated in Tiled cuts flipped or rotated. Its `Kind` is `twod.TiledMask`: its `TileMap()` is nil, it is not among `TileLayers()`, and it is never drawn.

A tile is made in the order recolour, mask, tint. The nearer setting wins: an object's recolour or mask over its layer's or its nearest group's, a layer's over the tile's own. Tints multiply, and a layer's `tint` multiplies Tiled's own layer tint. A layer's recolour leaves a tile holding none of its colours as it is, the stone above, and is refused only when it recolours none of the layer's tiles.

A name the loader isn't given fails the load, with the nearest one it holds; so do a mask layer's `cuts` naming no layer and a mask cell's tile with no `name`:

```
layer floor: property recolour: palette drak is not given; the nearest is dark
```

A recolour or a mask named with no `twod.Palettes` or `twod.Masks` given fails too, and so does any of the three on an object that isn't a tile.

A mask changes what is drawn, never what blocks or casts: cells keep their tile ids, so [collision](09-collision.md) and [light](19-lights/03-shadows.md) read the uncut tile. A mask may hold partial alpha, where a half-alpha pixel leaves the tile's pixel at half. `Set` and `SetFlip` on a cell keep its layer's recolour and its cut.

Tiled shows the properties, and through the [project file](12-classes-and-properties.md#in-tiled) as typed fields, but draws the tiles as they are: no recolour and no cut is previewed there, and a mask layer shows its mask tiles. The game is where the result is seen. The [tileset tool](19-lights/03-shadows.md#in-tiled) leaves the three out of its panel, having no palettes or masks to show them with, and refuses a tileset whose tiles name a palette or a mask.

Every variant is made once, at load: a frame costs nothing more. A frame's draw of a 40 × 109 map in an 800 × 600 view takes 47.8 µs, and 47.5 µs with every cell recoloured and cut. The load pays instead: the fixture above loads in 110 µs, 37 µs with its variants left out.

## Try it

[43-recolours-and-masks](../../examples/43-recolours-and-masks)

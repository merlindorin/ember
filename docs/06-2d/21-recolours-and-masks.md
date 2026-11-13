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

## Try it

[43-recolours-and-masks](../../examples/43-recolours-and-masks)

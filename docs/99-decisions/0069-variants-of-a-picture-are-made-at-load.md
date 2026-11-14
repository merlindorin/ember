# 0069. Variants of a picture are made at load

- Status: Accepted
- Date: 2026-11-13
- Area: rendering
- Owner: user

## Context

Games baked variants of pack art into new pictures with one-off generators (Lantervale's dark water, its slimes in
three colours): a program, its tests and a set of pictures to keep in step with the pack for each variant, when each
variant is the same picture through a recolour or a mask ([masks and recolours](../design/masks-and-recolours.md)).

## Decision

- A **recolour** swaps exact colours by a `twod.Palette`: `NewPalette(Swap(from, to)…)` in code or data, or
  `LoadPalette` from a picture of two rows of swatches. Colours match as 8-bit sRGB, alpha kept; other colours stay as
  drawn. A swap matches its colour at any alpha, or with `SwapAt(from, alpha, to)` (a half-clear swatch in a palette
  picture) only at that 8-bit alpha, which wins over the swap at any. `Recolour(tex, pal)` gives a new texture, and
  `LoadTexture(…, Recoloured(pal))` recolours one as it loads. A palette that matches no colour of the picture is an
  error.
- A **mask** keeps a picture only where a mask picture is opaque:
  `Masked(tex, region, mask, maskRegion, Inverted()…)`.
- Both run once, on the CPU, and give a `twod.Texture`, cached by its source and its operation: two equal asks share
  one texture, and a frame costs nothing more. They compose as recolour, then mask, then the draw's tint.

## Alternatives

- **On the GPU**, a palette lookup or a mask per draw: a variant per entity for free, but a lookup in every 2D pixel or
  a second shader that breaks batches, for sheets of a few kilobytes.
- **Masks as layer passes** (`AsMask`, `MaskedBy`): right for a moving mask, a pass per masked layer every frame for
  static tiles.
- **Keep baking** with one shared generator: still pictures to keep in step and a program to run for a shade.

## Consequences

A variant is the operation, kept in data, not its output. A recoloured sheet takes the memory its baked picture took,
and a recolour of a 384×48 sheet takes about 0.2 ms at load. A variant made from a texture changed later (`Touch`) is
not remade.

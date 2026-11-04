# 0034. Every tile draws whole

- Status: Accepted
- Date: 2026-10-30
- Area: rendering
- Owner: engine

## Context

Art must never be resized, cropped or sliced, yet [0029](0029-a-flat-tile-draws-its-ground.md) drew a flat tile's bottom only, and a tileset was one grid of one size. Wall sets made in Tiled are collections: pieces of their own sizes, often rectangles of a shared sheet, mixed with a grid of floor on one layer.

## Decision

- Every tile draws whole, at one scale per tileset (the grid's tile fits the map cell's width), its bottom-left on its cell's: a tile taller or wider than the grid rises above it or spreads to its right. A Tall tile still stands as a sprite, at its cell's bottom-centre.
- A tileset can hold tiles of their own, each a rectangle of any texture at any size: `LoadTiledTileset` reads Tiled collections (one picture per tile, or a rectangle of a shared one), and `MergeTilesets` puts several tilesets together, numbering each on from the last.
- A Tiled layer may draw from several tilesets: it gets the map's tilesets merged, numbered by global id.
- Tiles carry typed properties (`Properties(id)`) and are found by their `name` property (`ID(name)`).
- Maps draw the cells whose tiles rise or spread into view, not only the cells in it.

## Alternatives

- **Keep 0029's ground slice:** cuts art, and gives no way to place pieces of other sizes.
- **One map per tileset, stacked:** every layer split by sheet, and their tiles no longer sort together.
- **Scale each tile to its cell:** resizes art.

## Consequences

Supersedes 0029: a tileset cut taller than the map cell now shows its whole picture, so art meant to lie flat must be cut to the cell. Maps cull with a margin as wide as the largest tile.

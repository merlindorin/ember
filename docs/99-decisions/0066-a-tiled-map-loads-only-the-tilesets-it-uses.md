# 0066. A Tiled map loads only the tilesets it uses

- Status: Accepted
- Date: 2026-11-10
- Area: assets
- Owner: engine

## Context

A level can attach a whole library of tilesets so its designer picks pieces in Tiled; Lantervale's attaches 4,466
pictures, and `twod.LoadTiled` loaded every one of them.

## Decision

- `LoadTiled` decodes every tile layer and reads every tile object's id before loading anything, and loads the
  pictures of only the tilesets those take a tile from. That is the default; there is no option.
- The others are still read: their properties, collision shapes and image file names (`Tileset.ImageFile`), so tools
  and checks see them. They have no pictures.
- `Tiled.UsedTilesets()` lists the loaded ones; `Tiled.Tilesets()` still lists them all, in order.

## Alternatives

- An opt-in option (`OnlyUsedTilesets()`): every map without it pays for unused pictures, and no map wants them.
- Lazy pictures per tile, loaded when first drawn: draws would decode PNGs mid-frame, and a missing picture would show
  as an error in play rather than at load.
- Dropping unused tilesets from `Tilesets()`: the list would no longer match the map's order and global ids.

## Consequences

A map costs only what it places. A game that draws a tile from a tileset its map never places (a cell set at run time
to an unplaced tileset) loads that tileset itself, with `LoadTiledTileset`.

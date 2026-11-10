# 0067. A Tiled map reads only the tilesets it uses

- Status: Accepted
- Date: 2026-11-10
- Area: assets
- Owner: engine

## Context

[0066](0066-a-tiled-map-loads-only-the-tilesets-it-uses.md) skipped the pictures of unused tilesets but still read every
tileset's .tsj, so Lantervale had to embed its 2.3 MB library of tilesets just to load the crypt.

## Decision

- `LoadTiled` finds the used tilesets from the map alone: each tileset's global ids run from its `firstgid` to the
  next one's, and a tile layer's cell or a tile object's gid in that range uses it.
- An unused tileset of its own file (.tsj) is not read at all: its file and pictures needn't be in the file system.
  `Tilesets()` keeps it in its place as an empty `Tileset` that knows its `Source()`, so the list still matches the
  map's order and global ids.
- An unused tileset embedded in the map keeps its properties and shapes, which cost nothing to read, and loads no
  pictures. A used one is read and loaded as before, and a missing file is an error.
- `UsedTilesets()` lists the loaded ones. This supersedes 0066.

## Alternatives

- Reading unused .tsj files for their properties (0066): the game ships a whole library it never draws.
- A nil entry in `Tilesets()` for an unread tileset: every caller ranging over the list would have to check for nil.
- Dropping unread tilesets from `Tilesets()`: indices would no longer follow the map's tilesets.

## Consequences

A level costs only what it places, files included. Tools that want an unused tileset's tiles read its `Source()` with
`LoadTiledTileset`.

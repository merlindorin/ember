# 0029. A flat tile draws its ground

- Status: Superseded by [0034](0034-every-tile-draws-whole.md)
- Date: 2026-10-30
- Area: rendering
- Owner: architect

## Context

Wall sets like 0x72's cut their pieces taller than a map cell (16×48 on a 32×32 cell). A tile that wasn't Tall drew its whole tileset cell squashed into the map cell, so flat wall tops, edges and corners couldn't lie flat, and every piece had to be Tall.

## Decision

- A tile that isn't Tall draws only the bottom of its tileset cell, one map cell's worth: the cell's width and as much height as the map cell's shape asks of it, at the map cell, never squashed.
- A Tall tile still stands its whole picture on the cell, as before.
- Both anchor to the cell's bottom, so one tileset serves flat and standing pieces; a tileset whose cells match the map cell draws exactly as before.

## Alternatives

- **Per-tile regions,** `TileRegion(id, rect)`: more explicit, but a second way to cut a sheet beside `Cell`.
- **Squashing, as before:** the art can't be used flat.

## Consequences

Tall wall art lies flat where it isn't Tall. A tileset meant to be squashed must now be cut to the map cell's shape.

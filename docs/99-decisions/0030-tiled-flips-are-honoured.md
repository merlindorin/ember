# 0030. Tiled flips are honoured

- Status: Accepted
- Date: 2026-10-30
- Area: assets
- Owner: architect

## Context

Tiled stores a cell's flips, across, upside down and across the diagonal (which with the others makes the 90° turns), in the top bits of its tile id. LoadTiled dropped them, so a room drawn in Tiled with turned pieces looked different in the game.

## Decision

- LoadTiled decodes the flip bits apart from the tile id, and a TileMap keeps them per cell as a `TileFlip` (`TileFlipX`, `TileFlipY`, `TileFlipDiagonal`), read with `Flip(x, y)` and set with `SetFlip`.
- Flipped cells draw as Tiled draws them: the diagonal first, then across, then upside down.
- What a tile is (solid, tall, its properties) stays per tile id; the flip changes only how it looks.
- A Tall tile, which stands as a sprite, takes the across and upside-down flips; the diagonal can't apply to a standing picture and is left out.

## Alternatives

- **Dropping flips silently,** as before: the map in the game isn't the map that was made.
- **Refusing flipped maps** with an error: honest, but designers would have to cut a piece for every orientation.

## Consequences

Maps look in the game as they do in Tiled, and one piece serves every orientation; a tall piece meant to stand turned needs its own art.

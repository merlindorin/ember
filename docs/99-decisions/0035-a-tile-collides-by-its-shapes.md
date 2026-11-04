# 0035. A tile collides by its shapes

- Status: Superseded by [0036](0036-a-solid-tile-blocks-its-cell-for-paths.md) for paths; its shapes for bodies stand
- Date: 2026-10-30
- Area: core
- Owner: architect

## Context

Wall packs draw side walls as thin bars at a cell's edge. Solid tiles stopped bodies at their whole cell, so a bar blocked its full cell, and rooms had to be drawn with the wrong, full-width pieces.

## Decision

- A tile can carry collision shapes, rectangles in its pixels: `twod.Collision(id, rects…)`, `Tileset.Collision(id)`, read from Tiled's per-tile collision editor (polygons and ellipses by their bounds).
- Maps' `SolidIn` returns those shapes in world points, where the tile is drawn; a solid tile with no shapes is still its whole cell.
- A map's `Solid(x, y)`, which paths read, is true only when the cell is blocked whole: a solid tile with no shapes, or shapes covering the cell.

## Alternatives

- **Half-cell grids:** finer cells everywhere, for every map, to serve a few pieces.
- **Full-width wall pieces:** the pack's thin bars can't be used as drawn.

## Consequences

Bodies slide along a bar while paths go through its cell. A cell partly blocked is open to paths, so a route may brush a bar; bodies' own collision keeps them out.

# 0032. Lights see exact visibility polygons

- Status: Superseded by [0037](0037-lights-read-the-light-maps.md)
- Date: 2026-10-30
- Area: core
- Owner: architect

## Context

Fog of war and lights need what a point sees of a grid, with shadows that follow walls exactly; sampled rays gave ragged edges and stopped short of walls.

## Decision

- `nav.NewVisibility(grid)` is kept per light, and `Compute(from, radius)` gives the exact visibility polygon as a fan: the light first, then the edge in angle order, for `Canvas.TrianglesColored`. `Edges()` are the lit parts of solid cells' faces, so a game lights the walls whose foot it sees.
- Rays go from the light just either side of every lit corner of the solid cells within reach, merged along each wall side, plus a ring of rays for the reach's circle. Each marches through the cells as `Map.Clear` does, so a shadow starts exactly at a cell's edge and light never slips through two cells touching corner to corner. A light in a solid cell sees as from an open one.
- The walls and cells around the light are gathered again only when the light changes cell, its reach changes or the grid's `Version` does; a Compute then allocates nothing and takes about 0.14 ms for a 10-cell reach in a dungeon.

## Alternatives

- **A function, `nav.Visibility(grid, from, radius)`:** simpler to call, but it would gather and allocate on every call; a light computes every frame, so it keeps its own state.
- **Testing each ray against every wall segment:** the textbook sweep, but it cost 1.6 ms in a dungeon, against 20 steps a ray through the cells.
- **Sampled rays at fixed angles:** cheap, but shadows aren't exact and corners are missed.

## Consequences

Lights, fog and line of sight share one exact answer. The polygon and edges are the Visibility's own buffers, valid until its next Compute, so a game copies them to keep one, such as a static light's.

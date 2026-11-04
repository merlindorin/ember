# 0040. Routing follows where a body fits, not whole solid tiles

- Status: Accepted
- Date: 2026-10-31
- Area: core
- Owner: architect

## Context

By [0036](0036-a-solid-tile-blocks-its-cell-for-paths.md) any solid tile blocked its cell for paths. Once side bars reached 9 art px of 16 to keep sprites off them, the horde's flow field stopped a cell short of every wall all round a room: the cells along the walls are mostly open floor, so monsters there got no way and walked straight.

## Decision

- A flow field built with `nav.Body(w, h)` over a `nav.Roomy` grid asks the grid where a body's centre fits in each cell near a solid. A cell with room is routable, and its way starts from that point, the one nearest the cell's middle. Fully solid cells (faces, column bases, the chest) stay blocked.
- A step to or from a partly filled cell counts only if the body walks straight from point to point clear of every shape (`Roomy.Pass`), so no way crosses a bar or cuts a corner through a shape.
- Lines of sight still stop at any solid tile.
- `twod.TileArea` is Roomy. Each tile's shapes are read once per cell size, and the area keeps a copy of its tiles. The field keeps what it read of the grid until the grid or its version changes, so a new goal reads none of it.
- The horde routes for `game.EnemyBody`. `Find` and `Route` (A*) keep 0036's whole cells.

## Alternatives

- Keep 0036: the field stays a cell short of every wall, and the user marked that up as missing.
- A finer grid (half or quarter cells): exact, but four to sixteen times the field's cost, and every consumer of cells changes.
- Shrink the shapes back: the sprites draw over the bars again (3f0401bf).

## Consequences

A cell-change frame costs about 0.05 ms, against 0.15 ms before. Reading the area again, on a re-cut or a map edit, costs about 1.6 ms. Monsters route along walls, and through gaps a body fits, such as between stacked column bases. Chasers following `Route` and the horde's arrows may differ near walls. A map may later carry an optional `walk` layer (cells, or rectangles like `rooms`) that adds or removes routable cells over the computed rule. It isn't built; the dungeon designer draws one only if the computed field ever disagrees with what the user wants.

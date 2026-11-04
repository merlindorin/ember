# 0036. A solid tile blocks its cell for paths

- Status: Superseded by [0040](0040-routing-follows-where-a-body-fits.md)
- Date: 2026-10-30
- Area: core
- Owner: architect

## Context

[0035](0035-a-tile-collides-by-its-shapes.md) left a cell open to paths unless its tile's shapes covered it. A thin side wall is a room's boundary, so its cell open to paths showed every flow field a gap through every side wall: monsters outside walked into the bar and stuck there instead of going round to the door.

## Decision

- A map's `Solid(x, y)`, which paths read, is true for any solid tile, whatever its collision shapes cover; a tile with shapes is solid.
- Shapes stay for bodies only: `SolidIn` returns them, so a body inside walks right up to the bar.

## Alternatives

- **0035's rule, open unless covered:** paths through every thin wall.
- **A property for walkable-but-blocking tiles:** no tile needs it yet.

## Consequences

Supersedes 0035's paths bullet. Paths keep a cell's width from a bar that bodies may stand next to; no tile can be walkable for paths and still stop bodies.

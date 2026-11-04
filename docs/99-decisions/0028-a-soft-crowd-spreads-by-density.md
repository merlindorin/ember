# 0028. A soft crowd spreads by density

- Status: Accepted
- Date: 2026-10-30
- Area: core
- Owner: architect

## Context

Chasing enemies piled on top of each other. Making them collide with each other cost 32 ms per fixed step at 20,000 (bench2d), and jams them anyway.

## Decision

- `collide.Soft(push)` makes a body part of a crowd (getter `Soft()`, setter `SetSoft`). Soft bodies never sweep against each other, but still sweep exactly against solids and hard bodies, such as the hero, walls and doors, which they never push.
- The crowd is a grid with cells twice the size of the largest soft body, rebuilt once per step with no allocation. Each cell counts how much of it soft bodies cover.
- A soft move drops the part of its motion that would bring a cell over capacity. Once a step, the Tracker pushes each soft body down the slope of the overflow (and out of the middle of an over-full cell), capped per step. Each push slides against solids, and it may enter a cell with room or one emptier than the cell it leaves, so overflow drains to the edge of the crowd.
- The order is fixed and nothing iterates a map, so the same steps give the same positions.

## Alternatives

- Pushing overlapping pairs apart once a step: a pile held up only by overlap pushes compresses with depth, like water, so 20,000 bodies chasing one point crushed without bound (7,000 in one cell, 440 ms a step), whatever the push.
- Pairs with half-size cores that soft bodies sweep against: exact and bounded at 4× density, but 11 ms a step spread out and 54 ms piled.

## Consequences

20,000 soft bodies cost about 2.2 ms of CPU per step, spread out or piled, and their density stays under capacity × 1.5. Contact between soft bodies is statistical: they may overlap within a cell, up to its capacity.

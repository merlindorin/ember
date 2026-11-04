# 0024. A horde follows one flow field

- Status: Accepted
- Date: 2026-10-30
- Area: core
- Owner: architect

## Context

Enemies steering straight at the hero slid along a solid until they faced him head on and stopped there, and a survivors horde is thousands of them, so the way round must cost little per enemy.

## Decision

- `nav.NewFlowField(grid, opts…)` points every cell the way to one target, made by a Dijkstra from the target over the grid with the moves `Find` allows: diagonals never cut a corner.
- `SetTarget(at)` makes it again only when the target changes cell, the grid's `Version` changes, or, with `Around(area, columns, rows)` over an endless grid such as `InfiniteMap.AreaAround`, the target strays from the window's middle half and a new window is cut.
- `Toward(at)` is O(1): a unit way blended from the four cells around `at`, so walkers turn smoothly; `ok` is false outside the field, on solids and where the target can't be reached, and the game falls back to its own way.
- A 128×128 field is made in under 1 ms (a bucket queue on integer costs, a padded grid, solids read a chunk at a time), and `Toward` takes about 15 ns.

## Alternatives

- **`nav.Find` per enemy:** exact paths, but thousands of A* searches a second, each re-planned as the hero moves.
- **Local sliding,** turning along the hit normal when blocked: no grid, but enemies still trap in concave corners and pick sides at random.
- **Steering behaviours** (separation, avoidance): good for spacing, not for finding a way round walls; they can come on top of the field.

## Consequences

Chasing becomes one field a frame at most and one lookup per enemy. A field covers a window of the world, so enemies outside it fall back to straight lines until the window follows them.

## Addendum: straight where the target is in sight (2026-10-30)

An 8-way field gives only 8 ways, so in the open every chaser fell onto the same axis or diagonal and a horde ran behind the hero in single file. The field now also marks the cells with a clear straight line to the target, corner-safe like `Map.Clear`, ring by ring out from it, erring towards unseen at obstacle edges; from those cells `Toward` aims straight at the target, and the others keep the field's way, blended at the shadow's edge. The API is unchanged, and a 128×128 field is still made in under 1 ms. The alternative, a ray per cell, cost several times more for the same result.

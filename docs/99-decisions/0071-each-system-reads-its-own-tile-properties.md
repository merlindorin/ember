# 0071. Each system reads its own tile properties

- Status: Accepted
- Date: 2026-11-13
- Area: assets
- Owner: user

## Context

twod draws, yet its Tiled loader named six properties and its `Tileset` stored what they meant for collision and
light, read cell by cell through maps keyed by tile id: one light's reach cost 20 µs and 138 allocations a frame.

## Decision

- twod names only `tall`. A tile keeps its properties (`Tileset.Value`) and its shapes with their class
  (`Tileset.Shapes`); `Version`, `Cached`, `Place`, `MaxID` and the maps' `CellOrigin` serve the systems.
- `collide/tiles` declares `solid` and the shape class `body`: `tiles.Of(map)` is a `collide.Solid` and a `nav` grid.
  `collide.SolidAppender` (`AppendSolidIn(dst, area)`) lets a space ask into a slice it keeps, so a move among tile
  maps allocates nothing.
- `twod/light` declares `shadow`, `occluder_group` and `height`, and finds the World's tile maps itself: a map casts
  in the window it is in, hidden or not. twod's lit tiles get their silhouettes through `internal/lit`. A collide
  solid reading a map is marked by a method returning `lit.FromMap`, which only engine packages can name, and light
  skips it. `light.NoShadows()` keeps a map drawn as a picture from casting; `light.TileShadowsIn` gives what a map's
  tiles cast, without a window, for a game's rules that follow light. `emissive` is no longer declared: no tileset
  set it, sprites glow with `light.Emissive`, and tile glow had no user.
- Each system keeps a table per tileset, rebuilt by `Version`, and the rects of each map placed once, rebuilt lazily
  at the next query after the map changes, as a whole (a `Set` then one light's reach on a 40×109 map: 81 µs, 22.5 of
  them the `Set` itself), by chunk for an endless map.

## Alternatives

- Keep the names in twod with typed fields: a game without lights still loads them, and every new property edits
  the drawing package.
- Merge runs of whole cells into one caster: fewer casters, but soft shadow edges move at the seams; left out until a
  bench asks for it.

## Consequences

A map in a collide space but in no window no longer casts: light reads its World, not the space. Shadows and solids
outside the area asked are no longer returned, and flipped tiles whose shapes spill past their cell are no longer
missed when an area is asked alone. Games compose `tiles.Properties()`, `light.TileProperties()` and their own
classes on the loader, and pass `tiles.Of(map)` to collide and nav.

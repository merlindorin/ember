# Collision

Walls that stop the hero, and a plate that opens a door when he steps on it. Here a walled garden and a stone room stand side by side, with a wall between them; walking right, the hero crosses the sand plate in the garden, the door in the wall opens, and he walks into the room.

![The hero walks onto the plate, the door opens, and he walks through](collision.gif "ember · collision")

## The room

The room is a tile map like any other, from a picture in text. `Solid(wall)` on the tileset makes every wall tile stop things, each blocking its whole cell.

A tile can instead collide by shapes of its own: `twod.Collision(bar, rects…)` gives it rectangles in its pixels from its top-left, such as a thin wall bar, and `LoadTiledTileset` reads them from Tiled's collision editor (polygons and ellipses by their bounds). A body then stops at the bar where it is drawn and slides along it. A shape may reach past its cell, as a wall's top often does, and the map finds it from the neighbouring cells too. For `Find` and `Route`, the cell of any solid tile is blocked whole, so their paths never go through a thin wall's cell; a flow field given a body's size routes through what's left of it, as below. `tileset.Collision(id)` reads the shapes back, and `Solid(id)` is true for a tile with shapes too ([decision 0035](../99-decisions/0035-a-tile-collides-by-its-shapes.md)).

[room.go](../../examples/27-collision/room.go)

## Bodies and the space

`collide.NewSpace` holds what can collide, and `AddSolid` puts the walls in it. The plate is an invisible sprite with a body that is a `Trigger`: it stops nothing, but tells when something enters it. `collide.NewTracker` is the system that notices those entries every frame.

[main.go](../../examples/27-collision/main.go)

## The hero's body

A body is a child of what it belongs to. The hero's is a box at its feet, as wide as its legs and a little deep, so it bumps into a wall with its feet and can still pass in front of it.

[hero.go](../../examples/27-collision/hero.go)

## Moving

`space.Move` moves the hero by as much as it can: it stops at a wall and slides along it, so walking diagonally into a wall keeps going sideways. A body put down overlapping a solid, spawned or teleported into one, can slide along it or step out, but a move never takes it deeper. `space.Clear(from, to)` asks whether a straight line between two points crosses no solid, bodies aside: whether a wall stands between an enemy and the hero it means to hit, for instance.

[walk.go](../../examples/27-collision/walk.go)

## The door

When something enters the plate, the tracker emits a `collide.BodyEntered` event, read like any other. `Door` then turns the door's wall tile into floor with `Set`, and since floor isn't solid, the way is open.

[door.go](../../examples/27-collision/door.go)

## A crowd's bodies

A horde of hundreds that all collide with each other jams: each blocks the next, and they pile up on one spot. `collide.Soft(push)` makes a body soft instead. Soft bodies pass through each other as they move, `Move` reporting no hit between two of them, but still stop exactly at solid tiles and at bodies that aren't soft, such as the hero or a door, which they never push. The tracker keeps them spread: a soft body moving into a crowded spot is held back along that axis, and once a fixed step the tracker eases over-full spots outward, sliding along walls, as fast as `push` says, from 0 to 1. Contact between them is approximate, overlapping a little in a crowd, and deterministic: the same steps give the same positions. Twenty thousand soft bodies cost about 2 ms a step ([decision 0028](../99-decisions/0028-a-soft-crowd-spreads-by-density.md)).

```go
enemy.Add(collide.NewBody(collide.Box(16, 10), collide.Group(foes), collide.CollidesWith(foes), collide.Soft(1)))
```

`body.Soft()` reads it, and `SetSoft(0)` makes a body solid again.

## Pathfinding

`ember/nav` finds ways round the solid cells of a grid, such as a tile map, in pure Go for games and servers alike. Paths move diagonally but never cut a wall's corner, and are pulled straight wherever the way is clear; `nav.NoDiagonals()` keeps them to the four straight directions.

For one walker, `nav.New(level)` is a map over the grid. `Find(from, to)` is the way as waypoints, and `ok` is false when `to` is solid or out of reach, the way then ending as near as it gets. `Route(from, to)` is a way to walk: each step, `route.Toward(at, reach)` is the direction to the next waypoint, moving on once within `reach` of it, and it plans again by itself when the grid changes or the way ahead closes. `Clear(a, b)` says whether the straight line between two points crosses no solid cell.

```go
route := level.Route(hero.Position(), click)
hero.Translate(route.Toward(hero.Position(), 4).Scale(speed * dt))
```

For a horde going to one place, a flow field leads every cell to its target at once ([decision 0024](../99-decisions/0024-a-horde-follows-one-flow-field.md)). `nav.NewFlowField(grid)` makes it, `SetTarget` points it at the target each frame and rebuilds it only when the target changes cell or the grid changes, and `Toward(at)` gives each walker a smooth unit direction in constant time, so a thousand enemies cost a lookup each. Where the target is in clear sight, `Toward` aims straight at it, so a horde in the open comes from wherever each one stands, spread like a crowd; only behind obstacles does it follow the field's way round, rather than falling into single file along the eight directions of the grid. It returns false off the grid, on a solid cell or where the target can't be reached; walk straight then. Over an endless map, `nav.Around(things.AreaAround, 128, 128)` makes the field a window of cells around the target, moved when the target strays from its middle half; a 128 × 128 field rebuilds in under a millisecond. With `nav.Background()`, `SetTarget` reads the grid at once and makes the new field on another goroutine; the next `SetTarget` waits for it and shows it, so `Toward` leads by the field made for the call before, exactly one call late and the same every run. The first field is made at once. It suits a big field retargeted every frame, taking the rebuild off the frame.

`nav.Body(w, h)` routes a field for a body of that size, over a grid that knows its tiles' shapes, a `nav.Roomy` such as an endless map's `AreaAround`. A solid cell whose shapes leave room for the body is passable, entered where the body fits nearest the cell's middle, and a step to or from a partly filled cell counts only if the body walks it straight clear of every shape; cells solid all over stay blocked, and lines of sight still stop at solid tiles. `field.Point(at)` is where the field routes a body through the cell under `at`. A game's own grid joins in by having `Room(x, y, body)`, where a body's centre fits in a cell, and `Pass(from, to, body)`, whether it walks straight between two points ([decision 0040](../99-decisions/0040-routing-follows-where-a-body-fits.md), superseding [0036](../99-decisions/0036-a-solid-tile-blocks-its-cell-for-paths.md)).

```go
field := nav.NewFlowField(nil, nav.Around(things.AreaAround, 128, 128), nav.Background(), nav.Body(16, 10))
```

```go
field := nav.NewFlowField(nil, nav.Around(things.AreaAround, 128, 128))

field.SetTarget(hero.Position())
for e := range enemies {
	dir, ok := field.Toward(e.Position())
	if !ok {
		dir = hero.Position().Sub(e.Position()).Normalize()
	}
	e.Walk = dir
}
```

## Try it

[27-collision](../../examples/27-collision)

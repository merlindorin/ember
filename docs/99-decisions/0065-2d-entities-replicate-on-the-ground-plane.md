# 0065. 2D entities replicate on the ground plane

- Status: Accepted
- Date: 2026-11-09
- Area: net
- Owner: architect

## Context

`net` replicated only entities with a `scene.Transform`, and measured interest on x and z, while a 2D game's entities
are `twod.Item`s placed on x and y. Lantervale's online spike needs its heroes and monsters replicated as they are.

## Decision

- `net.Replicated` is an entity with a `Network`, and a pose: a `scene.Transform`, or a 2D placement with the methods
  a `twod.Item` has (Position, Rotation, Scale, their setters, GlobalPosition). net panics when one with neither joins
  a World. A transform is used first when an entity has both.
- A 2D pose travels as the one wire `Pose` on the ground plane: (x, y) is (x, 0, y), the rotation a yaw, the scale
  (sx, 1, sy). Interest, smoothing, prediction and corrections run unchanged on it.
- `net.SnapDistance(d)` is how far one predicted step must go to be shown at once (a dash) instead of blended. Its
  default stays 1, a metre in 3D; a 2D game passes one in its own units.

## Alternatives

- **A separate 2D wire pose** (Vec2, angle, Vec2): smaller before any codec, but a second message type through
  snapshots, datagrams, interpolation and prediction. A compact codec can pack the plane's zeros later.
- **Importing twod in net:** a named `*twod.Placement` instead of methods, but twod reaches net through automation and
  node3d, so it is an import cycle.
- **A space option on the server:** one switch for all entities, but a game may replicate both kinds.

## Consequences

2D games replicate their own entities with no adapter. `Replicated` no longer promises `GetTransform`, so code that read
it through the interface reads it from its own type. Distances in `Interest` and `SnapDistance` are in the game's units.

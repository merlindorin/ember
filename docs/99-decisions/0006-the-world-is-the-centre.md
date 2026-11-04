# 0006. The World is the centre

- Status: Accepted
- Date: 2026-10-28
- Area: core
- Owner: user

## Context

The ECS came from a separate module behind an `app.App` wrapper, so every program juggled two objects, and the package name `ecs` mislabelled events and resources.

## Decision

- The World lives in the package `ember/world`: entities, systems, resources, events, the schedule, the clock, plugins and presenters.
- `ember.New(opts...)` returns a `*world.World`, and a program is one chain: `ember.New(…).AddEntities(…).AddSystems(…).MustRun()`.
- Systems receive a `world.Frame`. There is no `app` package.
- The ECS is the design behind the World, explained in the guide, not a name in the code.

## Alternatives

- Keep go-ecs separate: reusable alone, but every rough edge of the API sat on the seam between the two modules.

## Consequences

Game code reads in terms of the World. The old terrain package `world` became `terrain`.

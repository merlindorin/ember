# 0015. The engine exposes no resources

- Status: Accepted
- Date: 2026-10-28
- Area: api
- Owner: user

## Context

Games reached engine services by type, as in `world.Resource[platform.Window](f.World)`. The dependency was hidden, the type was the key, and a missing service failed at runtime.

## Decision

- `world.Resource` is for a game's own shared values. The engine never exposes its services through it, and a test enforces this.
- The World's own state is World API: `f.Time`, `w.Time()`, `w.Stats()`.
- Engine services are values built with constructors and options (`platform.NewWindow`, `render.NewEnvironment`, `net.NewClient`, `automation.New`…). The game adds them to the World or passes them to `ember.New`, and hands them to its systems. Defaults are created when none is given.
- Windows, the environment, gizmos and billboards are entities. The first window added is the primary.

## Alternatives

- Keep type-keyed resources: shorter setup, but every dependency stays hidden and is checked only at runtime.
- Named accessors such as `input.Of(w)`: more readable, but still a hidden global lookup.

## Consequences

Every dependency is visible in `main` and checked by the compiler, at the cost of a few lines of wiring. The `ember.WithTitle`-style window options move to `platform.NewWindow`.

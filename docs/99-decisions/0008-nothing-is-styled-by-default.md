# 0008. Nothing is styled by default

- Status: Accepted
- Date: 2026-10-27
- Area: rendering
- Owner: engine

## Context

The smallest program needed environment code to get a plain background, and its image was bloomed and vignetted.

## Decision

- A window clears with its background colour (`ember.WithBackground`).
- Bloom, vignette, fog and colour grading are opt-in, through `render.With*` options or the `render.Post` chain.
- A visual style is game code, composed from `render.Post`, never an engine option.

## Alternatives

- A styled default: prettier first screenshots, but every game has to undo the engine's taste.

## Consequences

The hello cube is just a cube. Games that want a look ask for it.

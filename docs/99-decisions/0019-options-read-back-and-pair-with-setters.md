# 0019. Options read back and pair with setters

- Status: Accepted
- Date: 2026-10-29
- Area: api
- Owner: architect

## Context

The second guide review found options without getters, setters whose arguments differed from their option, a `With` prefix on some packages only, and grids, repeats and events each named their own way.

## Decision

- Every option has a getter of the same name. A setter exists where a game is meant to change the value at runtime, and takes the option's arguments.
- Options have no `With` prefix, unless the bare name would clash with a type in the same package (`ember.WithTheme`, `twod.WithEffect`).
- One word per idea: `Cell(w, h)` describes every grid, `Loop()`, `PingPong()` and `Once()` every repeat, and events read `<Subject><Past>` (`AnimationFinished`, `TweenFinished`).
- Required data is positional (text, value, columns, textures), and flags are options.
- Shared values such as faces and easings are functions, not variables.

## Alternatives

- Setters everywhere: complete, but they promise runtime changes the engine doesn't support well.
- Keep `With` everywhere: unambiguous, but noisy in the chained style the examples use.

## Consequences

A reader can guess an API from one page. Adding an option means adding its getter, and its setter when change is meant.

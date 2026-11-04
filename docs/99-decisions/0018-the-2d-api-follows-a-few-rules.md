# 0018. The 2D API follows a few rules

- Status: Accepted
- Date: 2026-10-29
- Area: api
- Owner: architect

## Context

The guide review found the same words meaning different things across twod and ui: `At` placed a centre, a corner or an origin, `Size` meant five things, and children were added two ways.

## Decision

- `At` places an element's pivot. Sized 2D elements are centred by default, a ui element's Anchor is its pivot, and lines, polygons and tile maps use their own origin.
- `Add(children…)` is the one verb on every parent, and returns the parent.
- `Size(w, h)` is always a box in points. Other sizes have their own names: `FontSize`, `CellSize`, `Measure`.
- `Color` is what an element draws with, and `Tint` multiplies it and its children.
- Loaders are `LoadX(fsys, name, opts…)` and `NewX(data, opts…)`. An option and its setter pair (`Still`/`SetStill`), and there is one way to do each thing (`Strip` cuts every sheet).

## Alternatives

- Per-type conventions: each type reads well alone, but readers must relearn every page.

## Consequences

A new 2D or ui API is checked against these rules before it lands.

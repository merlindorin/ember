# 0037. Lights read the light maps

- Status: Superseded by [0042](0042-2d-lights-and-shadows-on-the-gpu.md)
- Date: 2026-10-31
- Area: core
- Owner: architect

## Context

[0032](0032-lights-see-exact-visibility-polygons.md) gave lights and sight exact visibility polygons from `nav.Visibility`. The game moved to light maps drawn by area, with a soft disc for the hero's torch, and nothing calls the polygons any more.

## Decision

- Lights and sight read the light maps: each light lights its drawn area, joined through open doors, and walls stop it.
- `nav.Visibility` and its polygons are removed from the engine.

## Alternatives

- **Keep `nav.Visibility` for later:** dead API to maintain and document, with no caller.

## Consequences

Supersedes 0032. Shadows follow areas and walls by cell, not exact polygons; a game that wants exact visibility writes its own.

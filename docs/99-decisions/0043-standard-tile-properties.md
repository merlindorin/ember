# 0043. Standard tile properties, checked by the loader

- Status: Accepted
- Date: 2026-11-01
- Area: assets
- Owner: user

## Context

Tiled tiles carry free-text custom properties. The loader read four of them by ad-hoc rules, the demo reparsed its own, and nothing caught a misspelled name or a wrong type (`docs/design/tiled-tile-properties.md`).

## Decision

- One documented set of engine tile properties, typed, with defaults: `solid`, `tall`, `shadow` (enum `cast`/`none`), `shadow_group` (0–2), `height`, `emissive` (`twod.EngineTileProperties`).
- A game declares its own with `twod.TileProperties(defs…)` on `LoadTiled`/`LoadTiledTileset`: name, kind, default, enum values or range.
- The loader checks every known property: a wrong kind, an enum value or a number out of range is an error naming the tileset, the tile and the property. Unknown ones are kept; `twod.NearMiss` flags likely misspellings.
- Values read typed with `ts.Value(id, name)`, defaults included.
- `twod.TiledProject(defs…)` writes Tiled's custom types (checked in as `twod/tiled/ember.tiled-project`), and the tileset tool builds its tile panel from the same definitions.

## Alternatives

- **Keep ad-hoc names:** misspellings stay silent and every game reparses JSON.
- **Tiled classes only:** strong types in Tiled, but hand edits and older files break, and the loader must read plain properties anyway.

## Consequences

A tileset with a wrong engine property now fails to load instead of half-working; the demo keeps `Property` (text) until its own properties move to `Value`.

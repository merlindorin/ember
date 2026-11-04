# Tiled tile properties

Status: **built (2026-11-01), decision 0043 (3dd84d4d; the tool's panel 8feb20fb).** The demo's own properties move to `ts.Value` with the demo migration.

Tiles in Tiled tilesets carry custom properties. Today the loader reads four of them (`solid`, `tall`, `shadow`, `occluder_group`) with ad-hoc rules, and the demo reads its own (`name`, `reach_down`, `reach`, `cells`, `door`, `pair`, `light`) in its own code. Nothing checks a misspelled name or a wrong type. This note makes one documented set that the loader reads and checks, Tiled shows as typed fields, and the tileset tool edits from the same definition.

## Engine properties

Read by `twod.LoadTiled` and `twod.LoadTiledTileset`, on a tile:

| Property | Type | Default | Meaning |
|---|---|---|---|
| `solid` | bool | false (true when the tile has collision shapes) | blocks bodies, as its shapes or its whole cell |
| `tall` | bool | false | drawn from its foot, y-sorted, lit at its base (light) |
| `shadow` | enum `cast`, `none` | `cast` | `none`: blocks bodies, not light |
| `occluder_group` | int 0–2 | 0 | occluder group, for `BlockedBy` |
| `height` | float, points | the tile's picture height when `tall`, else 0 for a floor and its cell height for a solid | how tall it stands, for shadow length (light) |
| `emissive` | float, points | 0 | glows and lights round it over this reach (`light.Emissive`) |
| `animation` | Tiled's own animation frames | — | read as today |

In a tile's collision objects, a shape's class says what it blocks:

| Class | Bodies | Light | Engine |
|---|---|---|---|
| none | yes | yes | `twod.Collision` |
| `shadow` | no | yes | `twod.ShadowShapes` |
| `body` | yes | no | `twod.BodyShapes` (2026-11-02): a bar's reach past its art, a low rail |

The tile property `shadow: none` stays the switch for the whole tile: it silences every shape's light. A solid tile whose only shapes are `body` ones casts nothing, not its whole cell.

## Game properties

Every other property is the game's:
- it's kept, typed, and read with `ts.Value(id, name)`, which returns a `twod.Value` (with String, Int, Float, Bool, Color and Any, and ok). `ts.Property(id, name)` stays a plain string until the demo migration, since the demo calls it;
- the demo's `name`, `reach_down`, `reach`, `cells`, `door`, `pair` and `light` move to it in the demo migration, so the demo stops reparsing the JSON;
- a game declares its own with `twod.TileProperties(defs…)` on the loader, giving each a name, type, default and description. That definition is what the checks, the Tiled project file and the tool use.

## Checks

The loader returns an error that names the tileset, tile id and property for:
- a known property with the wrong type (`tall: "yes"`);
- an enum value out of range (`shadow: "half"`);
- a `occluder_group` outside 0–2.

An unknown property is kept, and the tool reports it once (in a debug log, or a warning list) as possibly misspelled, when it's within two edits of a known name (`occluder_grup`).

## Tiled project file

`ember tiled-types > ember.tiled-project`, or a checked-in `ember/twod/tiled/ember.tiled-project`, holds Tiled's custom property types: an `ember.Tile` class with the engine properties and their defaults, and the `shadow` enum. A game's own definitions are added to it. Opened as a Tiled project, it makes Tiled show the properties as typed fields with defaults, instead of free text.

## The tileset tool

The tool's tile panel is built from the same definitions:
- checkboxes for bools;
- choices for enums;
- number fields for ints and floats;
- read-only rows for unknown properties, flagged when they look misspelled.

The game's definitions come from a `-props` file (JSON) or the demo's built-in set.

## Map objects (later)

The same standardisation for objects placed in maps, the light sources: an object class `light` with `kind`, `radius`, `color`, `falloff`, `elevation`, `static` and `priority`. It's on record here and comes after tiles.

## Alternatives

- **Keep ad-hoc names:** misspellings stay silent, and every game reparses JSON for its own properties.
- **Tiled classes only:** strong types in Tiled, but older maps and hand edits break, and the loader must still read plain properties.

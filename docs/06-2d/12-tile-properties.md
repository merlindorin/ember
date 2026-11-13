# Tile properties

The properties a tile of a Tiled tileset can carry, set in Tiled's tileset editor or the engine's tileset tool: those the engine reads, and a game's own, typed and checked when the tileset loads ([decision 0043](../99-decisions/0043-standard-tile-properties.md)). A reference page: the tile maps that use them are on [Tile maps](08-tile-maps.md), [Collision](09-collision.md) and [Tiled maps](11-tiled.md).

## The engine's properties

`twod.LoadTiled` and `twod.LoadTiledTileset(fsys, name, opts…)` read these from each tile, and `twod.EngineTileProperties()` lists them:

| Property | Kind | Default | Meaning |
|---|---|---|---|
| `solid` | bool | false | blocks bodies, as its collision shapes or its whole cell; true when it has shapes |
| `tall` | bool | false | drawn from its foot, sorted by y with the characters, lit at its base |
| `shadow` | `cast` or `none` | `cast` | `none`: it blocks bodies but not light |
| `occluder_group` | int, 0 to 2 | 0 | its occluder group, for lights `BlockedBy` others |
| `height` | float, in its pixels | 0 | how tall it stands, for the length of its shadows; 0 is its picture's height when it's tall, its cell's when it's solid, and nothing otherwise |
| `emissive` | float, in its pixels | 0 | how far round it it glows: read, checked and returned by `ts.Emissive(id)`, but map tiles don't glow from it yet; a sprite glows with [`light.Emissive`](19-lights/06-shape-lights.md) |

The engine reads three Tiled classes:

| Class | On | Meaning |
|---|---|---|
| `ember.Tile` | a tile, declared in the [project file](#in-tiled) | Tiled shows the tile's properties as typed fields with their defaults |
| `shadow` | a collision object in a tile | blocks light only |
| `body` | a collision object in a tile | blocks bodies only |

A tile can carry a class of its own too, set on the tile in Tiled (its type before Tiled 1.9): `ts.Class(id)` returns it, empty when it has none. A collision object without a class blocks both: see [Shadows in Tiled](19-lights/03-shadows.md#in-tiled). The tileset reads the properties back with `ts.Solid(id)`, `ts.Height(id)`, `ts.Emissive(id)` and the like; lights use the height for their shadows' length.

## A game's own properties

Every other property is the game's, kept as it was set: a door's pair, a room's name. `twod.TileProperties(defs…)`, given to the loader, declares them, each a `twod.Property{Name, Kind, Default, Values, Min, Max, Doc}`: its kind `twod.KindBool`, `KindInt`, `KindFloat`, `KindString`, `KindEnum`, whose `Values` are the words it can take, or `KindColor`; a number's range when `Min` is under `Max`; and `Doc`, what it does.

`ts.Value(id, name)` reads a property, as set or else its declared default, as a `twod.Value`: `String()`, `Int()`, `Float()`, `Bool()`, `Color()`, from Tiled's `#aarrggbb`, and `Any()`. Its second result is false for a property neither set nor declared. `ts.Property(id, name)` gives the text alone.

```go
door := twod.Property{Name: "door", Kind: twod.KindEnum, Default: "closed", Values: []string{"open", "closed", "locked"}, Doc: "how the door starts"}
walls, err := twod.LoadTiledTileset(files, "walls.tsj", twod.TileProperties(door))
if err != nil {
	log.Fatal(err)
}
if v, ok := walls.Value(id, "door"); ok && v.String() == "locked" {
	lock(id)
}
```

## Checks

A property of the wrong kind, a word outside its enum's values, or a number outside its range fails the load, with an error naming the tileset, the tile, counted from 0 as Tiled does, where a `Tileset` counts from 1, and the property:

```
twod: walls.tsj: tile 3: property shadow: "half" is not one of cast, none
```

An unknown property is kept as it is; `twod.NearMiss(name, defs…)` returns the known name it's likely a misspelling of, within two letters, as `occluder_grup` is of `occluder_group`.

## In Tiled

`twod.TiledProject(twod.TileProperties(defs…))` writes a Tiled project file declaring the engine's properties and the game's as a class, `ember.Tile`, with an enum for each enum property. Open it as the project in Tiled and give the tiles the class `ember.Tile`: Tiled then shows their properties as typed fields with their defaults, `shadow` a list of its two values. The engine's alone are in [ember.tiled-project](../../twod/tiled/ember.tiled-project); `go run ./tools/tileset -tiled-project -props game.json` prints one with a game's too.

## In the tileset tool

The [tileset tool](19-lights/03-shadows.md#in-tiled) shows a tile's engine properties as Tiled names them, in a table of names and values: a switch for each bool, a pop-up for each enum and `occluder_group`, a field for each number, and the animation read only; an unset one shows its default with "(default)", muted, and the first change sets it. A set false is written to the file and kept. A misspelling of an engine property is flagged among them. A game's own properties show beside them when the tool is given them, `-props props.json`, a JSON list of `{name, kind, default, values, min, max, doc}`; without it they don't show, and are saved byte for byte.

![The tileset tool's tile 0: solid switched on, tall off with (default), shadow cast (default), occluder_group 0 (default), height 64, emissive 0, animation none](tile-properties.png "tileset · walls.tsj")

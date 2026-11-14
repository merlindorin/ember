# Classes and properties

The classes and properties a Tiled map can carry, set in Tiled or the engine's tileset tool: those the engine reads, and a game's own, typed and checked when the map loads ([decision 0043](../99-decisions/0043-standard-tile-properties.md), [decision 0070](../99-decisions/0070-tiled-classes-are-declared-in-go.md)). A reference page: the tile maps that use them are on [Tile maps](08-tile-maps.md), [Collision](09-collision.md) and [Tiled maps](11-tiled.md).

## The engine's properties

Each system declares the tile properties it reads, and reads them itself ([decision 0071](../99-decisions/0071-each-system-reads-its-own-tile-properties.md)): twod `tall`, `tint`, `recolour` and `mask`, which it draws by, `collide/tiles` what stops bodies, and `twod/light` what blocks light. A game gives the loader the declarations of the systems it uses, so their properties are checked and filled with their defaults:

```go
level, err := twod.LoadTiled(files, "level.tmj", tiles.Properties(), light.TileProperties())
```

| Property | Declared by | Kind | Default | Meaning |
|---|---|---|---|---|
| `tall` | twod, `twod.EngineTileProperties()` | bool | false | drawn from its foot, sorted by y with the characters, lit at its base |
| `solid` | collide/tiles, `tiles.Properties()` | bool | false | blocks bodies, as its collision shapes or its whole cell; true when it has shapes |
| `shadow` | twod/light, `light.TileProperties()` | `cast` or `none` | `cast` | `none`: it blocks bodies but not light |
| `occluder_group` | twod/light, `light.TileProperties()` | int, 0 to 2 | 0 | its occluder group, for lights `BlockedBy` others |
| `height` | twod/light, `light.TileProperties()` | float, in its pixels | 0 | how tall it stands, for the length of its shadows; 0 is its picture's height when it's tall, its cell's when it's solid, and nothing otherwise |
| `tint` | twod, `twod.EngineTileProperties()` | color | none | multiplies its colours and opacity when drawn; also on a tile object and a layer, where it multiplies Tiled's own tint; see [Recolours and masks](21-recolours-and-masks.md#in-tiled) ([decision 0069](../99-decisions/0069-variants-of-a-picture-are-made-at-load.md)) |
| `recolour` | twod, `twod.EngineTileProperties()` | string | none | the palette, given to the loader by `twod.Palettes`, its colours are swapped by; also on a tile object and a layer |
| `mask` | twod, `twod.EngineTileProperties()` | string | none | the tile of the mask tileset, given by `twod.Masks`, by its `name`, that cuts it; also on a tile object |

`twod.Declared(opts…)` lists the properties declarations give, as `[]twod.Property`. A tileset keeps every property as set and every shape with its class; `ts.Value(id, name)` reads a property and `ts.Shapes(id)` the shapes, each a `twod.TileShape{Rect, Class}`, and the systems read them through those. `tiles.Stops(ts, id)` says whether a tile stops bodies and `tiles.Rects(ts, id)` the shapes it stops them with, for code that asks a tileset without a map. A map that names a palette or a mask is loaded with them: `twod.Palettes(map[string]*twod.Palette)` and `twod.Masks(maskTileset)`, given to `LoadTiled`.

A tileset made in code sets them with each system's options: `twod.Tall(ids…)`; `tiles.Solid(ids…)`, `tiles.Collision(id, rects…)` and `tiles.BodyShapes(id, rects…)`; `light.TileShadow(on, ids…)`, `light.TileOccluderGroup(n, ids…)`, `light.TileHeight(pixels, ids…)` and `light.TileShadowShapes(id, rects…)`. `twod.TileValue(id, name, v)` and `twod.TileShapes(id, class, rects…)` set any property or shape, a game's own too.

## The engine's classes

| Class | On | Read by | Meaning |
|---|---|---|---|
| `ember.Tile` | a tile, declared in the [project file](#in-tiled) | Tiled | Tiled shows the tile's properties as typed fields with their defaults |
| `shadow` | a collision object in a tile | twod/light | blocks light only |
| `body` | a collision object in a tile | collide/tiles | blocks bodies only |
| `ember.Layer` | a layer, declared in the project file | Tiled | Tiled shows `tint` and `recolour` as typed fields |
| `ember.Object` | an object, declared in the project file | Tiled | Tiled shows `tint`, `recolour` and `mask` as typed fields |
| `mask` | a tile layer | twod | a mask layer: its tiles, from the mask tileset, cut the tiles under them in the tile layer its `cuts` property names, a sibling, else the map's one layer of that name; never drawn, its `Kind` `twod.TiledMask` |

A layer or an object of a game's class takes `tint`, `recolour` and `mask` too, added in Tiled by hand; its class's members stay the game's own. A collision object without a class blocks both: see [Shadows in Tiled](19-lights/03-shadows.md#in-tiled). A tile's own class, set on the tile in Tiled (its type before Tiled 1.9), reads with `ts.Class(id)`, empty when it has none.

## A game's own properties

Every other property is the game's, kept as it was set: a door's pair, a room's name. `twod.TileProperties(defs…)`, given to the loader, declares those every tile can carry, each a `twod.Property{Name, Kind, Default, Values, Min, Max, Required, Class, Doc}`: its kind `twod.KindBool`, `KindInt`, `KindFloat`, `KindString`, `KindEnum`, whose `Values` are the words it can take, `KindColor`, `KindFile`, a file's path in the map's file system, `KindObject`, an object's id, or `KindClass`, whose members are those of the declared class named by `Class`; a number's range when `Min` is under `Max`; and `Doc`, what it does.

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

## A game's own classes

Tiled saves only the members of a class that were changed; their defaults live in the project file, not in the map. So the loader has to know a class to read what a map left at its default, and a game declares its classes in Go: `twod.Classes(classes…)`, given to `LoadTiled`, each a `twod.Class{Name, On, Shapes, Properties, Doc, Color}`. `On` is what it is the class of, `twod.OnMap`, `OnLayer`, `OnObject` or `OnTile`, joined with `|` for several; `Shapes` are the shapes an object of the class may have, any when empty; `Color`, an ember `color.Color`, is what Tiled draws the class's maps, layers, objects and tiles in, Tiled's own grey when zero. Go is the source and Tiled the editor: a class added in Tiled's own type editor is overwritten when the project file is written, and refused at load.

```go
wall := twod.Class{Name: "wall", On: twod.OnTile, Properties: []twod.Property{
	{Name: "durability", Kind: twod.KindInt, Default: 100, Min: 1, Max: 1000, Doc: "how many hits it takes"},
}}
exit := twod.Class{Name: "exit", On: twod.OnObject, Properties: []twod.Property{
	{Name: "to", Kind: twod.KindString, Required: true, Doc: "the level it leads to"},
}}
level, err := twod.LoadTiled(files, "level.tmj", tiles.Properties(), light.TileProperties(), twod.Classes(wall, exit))
if err != nil {
	log.Fatal(err)
}
for _, o := range level.Objects("things") {
	if o.Type == "exit" {
		to, _ := o.Properties.String("to")
		addExit(o.Rect, to)
	}
}
```

An object's class is its `Type`, a layer's its `Class`, the map's `level.Class()` and a tile's `ts.Class(id)`. At load, a map, layer or object of a declared class has every property of the class: the file's value, else the default, members of a class-typed property included. A tile of a class reads its class's defaults when asked, through `ts.Value`, beside the properties every tile has. The three ways Tiled saves a class load the same.

## Checks

A property of the wrong kind, a word outside its enum's values, or a number outside its range fails the load, with an error naming the tileset, the tile, counted from 0 as Tiled does, where a `Tileset` counts from 1, and the property:

```
twod: walls.tsj: tile 3: property shadow: "half" is not one of cast, none
```

Once a class is declared for a use, a map, layer, object or tile of that use is checked too: a class that isn't declared, with the declared name it's likely a misspelling of, a class used where it doesn't belong, a property the class doesn't have, a value of the wrong kind or out of range, a `Required` property left out or empty, and an object of a shape the class doesn't take fail the load. With nothing declared for a use, nothing of it is checked.

An unknown tile property is kept as it is; `twod.NearMiss(name, defs…)` returns the known name it's likely a misspelling of, within two letters, as `occluder_grup` is of `occluder_group`.

## In Tiled

`twod.TiledProject(opts…)`, given the same declarations as the loader, the systems' such as `tiles.Properties()` and `light.TileProperties()`, the game's `twod.TileProperties` and its `twod.Classes`, writes a Tiled project file declaring them: `tall`, `tint`, `recolour`, `mask` and every declared tile property as a class, `ember.Tile`, with an enum for each enum property; `ember.Layer` and `ember.Object` with the properties a layer and an object take; the mask layer's class `mask`; and every declared class, a tile class with `ember.Tile`'s members first. Open it as the project in Tiled and give the tiles the class `ember.Tile`, or one of the game's: Tiled then shows their properties as typed fields with their defaults, `shadow` a list of its two values. It is written as Tiled 1.12 writes a project. `twod.ProjectFolders(folders…)` sets the folders Tiled's project panel shows, relative to the project file, in the order given, `.` alone without it; the loaders ignore it. A game keeps its project file beside its maps and tests that it is what its declarations write, so the two never drift. The engine's alone, twod's, collide/tiles' and twod/light's, are in [ember.tiled-project](../../twod/tiled/ember.tiled-project); `go run ./tools/tileset -tiled-project -props game.json` prints one with a game's tile properties too.

## In the tileset tool

The [tileset tool](19-lights/03-shadows.md#in-tiled) shows a tile's engine properties as Tiled names them, in a table of names and values: a switch for each bool, a pop-up for each enum and `occluder_group`, a field for each number, and the animation read only; an unset one shows its default with "(default)", muted, and the first change sets it. A set false is written to the file and kept. A misspelling of an engine property is flagged among them. `tint`, `recolour` and `mask` are left out, as the tool has no palettes or masks to show them with, and a tileset whose tiles name a palette or a mask is refused. A game's own properties show beside them when the tool is given them, `-props props.json`, a JSON list of `{name, kind, default, values, min, max, doc}`; without it they don't show, and are saved byte for byte.

![The tileset tool's tile 0: tall off with (default), solid switched on, shadow cast (default), occluder_group 0 (default), height 64, animation none](tile-properties.png "tileset · walls.tsj")

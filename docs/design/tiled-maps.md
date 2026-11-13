# Tiled maps: the whole map, and nothing ignored silently

Status: **approved by the user (2026-11-13), with the answers under "Decided"; §6 was added after and is for the
user's review.** Built: step A (493b522e7, decision 0068), B (f59d7ebd7), C (628bbbda6, 4a7a75a79). Decision number reserved: 0068.
engine-render, who owns `ember/twod`, has read it against the code; its remarks are folded in.

## Why

`twod.LoadTiled` reads a narrow part of a Tiled map and hands the game less than the file holds. What it reads, it
reads correctly; the trouble is what it drops.

**A game has to parse its map twice.** Lantervale's `levels.LoadFS` calls `twod.LoadTiled`, then reads the same JSON
again with its own `header`, `rawLayer` and `rawObject` structs, because `twod.Tiled` does not say:

- the map's class, or its size in cells;
- the order of its layers: tile layers come back as one list and object layers in a map keyed by name, so how they
  interleave (`ground`, `path`, `walls`, `props`) is lost, and two layers of one name collide;
- an object's id, or whether it is visible.

**The map's author is not told what was dropped.** These are in the file, show in Tiled, and are ignored without a
word:

| In Tiled | Today |
|---|---|
| a layer's visibility, opacity, offset, tint, parallax | not read: a hidden layer still draws |
| an object's rotation and visibility | not read |
| a tile animation's frame durations | the first frame's duration is used for every frame |
| a polygon or an ellipse drawn on a tile | becomes its bounding rectangle |
| a property of a custom class, or one that points to an object | flattened to text |
| a text object, an object made from a template | read as an empty rectangle |

**These are refused with an error**, which is honest but narrow: group layers, image layers, infinite maps, staggered
and hexagonal maps, zstd compression, XML files (.tmx, .tsx).

Lantervale's two levels (`crypt.tmj`, `temple.tmj`) use none of these today. They are edited by hand in Tiled, so the
silent ones are the risk: a hidden layer or a turned prop looks right in Tiled and wrong in the game.

## The rule

Everything a Tiled map can hold is, in the loader, one of three things:

1. **Applied**: the engine does what Tiled shows.
2. **Exposed**: the value is on the API for the game to use, and the doc says the engine does not apply it.
3. **Refused**: `LoadTiled` returns an error naming the file, the layer and the object.

Nothing is dropped silently. The one exception is what only the editor uses and has no meaning in a game (terrain
and Wang sets, export settings, the editor's own state): those are listed in the package doc as not read.

A test holds the rule: it loads a map that sets every field the loader knows, and fails when the JSON holds a key
that is in none of the lists. The not-read list names the keys every export carries (`tiledversion`, `version`,
`nextlayerid`, `nextobjectid`, `compressionlevel`, `editorsettings`, `renderorder`, a layer's `locked`, a tile's
`probability`), so real maps pass. A key a newer Tiled adds fails the test: the lists grow on purpose, never by
default.

## 1. The whole map

`Tiled` says what the file says of itself, and keeps its layers in the file's order.

```go
func (t *Tiled) Class() string
func (t *Tiled) Columns() int            // the map's size in cells
func (t *Tiled) Rows() int
func (t *Tiled) TilePixels() (int, int)  // a cell in the map's pixels
func (t *Tiled) Scale() float32          // points per map pixel, from CellSize: objects are in map pixels
func (t *Tiled) Background() color.Color // the map's background colour, for a game's clear colour
func (t *Tiled) ParallaxOrigin() geom.Vec2

func (t *Tiled) Layers() []*TiledLayer             // the map's top layers, bottom first; a group holds its own
func (t *Tiled) Layer(name string) (*TiledLayer, bool)
func (t *Tiled) Walk(func(*TiledLayer) bool)       // every layer, groups and what they hold, in draw order
func (t *Tiled) TileLayers() []*TileMap            // every tile layer's map, in draw order: today's Layers()
func (t *Tiled) Object(id int) (TiledObject, bool) // by Tiled's object id
```

```go
type TiledLayer struct {
    ID         int
    Name       string
    Class      string
    Kind       TiledLayerKind  // TiledTiles, TiledObjects, TiledImage, TiledGroup
    Properties TiledProperties
    Visible    bool
    Opacity    float32
    Offset     geom.Vec2       // map pixels
    Tint       color.Color
    Parallax   geom.Vec2
    DrawOrder  TiledDrawOrder  // an object layer's: TiledTopDown (sorted by y) or TiledIndex (file order)
}

func (l *TiledLayer) TileMap() *TileMap      // a tile layer's map, else nil
func (l *TiledLayer) Objects() []TiledObject // an object layer's objects
func (l *TiledLayer) Picture() *Sprite       // an image layer's picture, else nil
func (l *TiledLayer) Layers() []*TiledLayer  // a group's layers, bottom first
func (l *TiledLayer) Layer(name string) (*TiledLayer, bool)
func (l *TiledLayer) Parent() *TiledLayer    // the group it is in, nil at the top
```

`TiledObject` gains `ID int`, `Visible bool` and `Rotation float32` (radians, clockwise, about the object's origin as
Tiled turns it).

**What changes for callers:** `Layers()` returns `[]*TiledLayer` instead of `[]*TileMap`, and `Layer(name)` a
`*TiledLayer`. `Objects(layer)` and `LayerProperties(name)` stay as shortcuts. The callers are example 29, the tileset
tool and three Lantervale packages; they are fixed mechanically in the same branch.

**Two layers of one name in the same group** (or both at the top) are an error naming both, since `Layer(name)`
could only answer for one. The same name in two groups is fine.

**A tile object as a sprite.** Every game that places pictures off the grid writes the same lines: region, size,
pivot, flip. The engine does it once, as Tiled draws it:

```go
// Sprite is tile object o's picture where it stands, in the map's points: its flips, its rotation and its size
// as Tiled shows them.
func (t *Tiled) Sprite(o TiledObject, opts ...SpriteOption) *Sprite
```

- **Anchor:** the sprite's pivot is the object's own anchor, the bottom-left corner unless the tileset's
  `objectalignment` says otherwise, and it turns about it. A pivot at the middle of the foot would put a turned prop
  half its width away. The y stays the foot's, so `SortByY` orders it as before.
- **Size:** a tile object resized in Tiled is drawn at the object's width and height, not the tile's. Today the
  resize is dropped.

## 2. Layer and object settings

| In Tiled | Becomes |
|---|---|
| layer hidden | **applied** to a tile layer's `TileMap` and an image layer's picture: not drawn. It still blocks bodies, paths and light (see below). **Exposed** as `Visible`. Its objects are still listed: a hidden `markers` layer is how data layers are usually kept |
| layer opacity, tint | **applied** to the `TileMap` or picture as its tint; **exposed**. A half-opaque layer still blocks in full |
| layer offset | **applied**: the `TileMap` is moved, and the layer's objects are returned already shifted; **exposed** |
| layer parallax | **exposed** only, with the map's `ParallaxOrigin`. twod's parallax is per twod layer, so two Tiled layers of different parallax go on two twod layers, which the game makes |
| object layer draw order | **exposed** as `DrawOrder`: whether Tiled sorts the layer's objects by y or keeps the file's order |
| object hidden | **exposed** as `Visible`; `Tiled.Sprite` returns it hidden |
| object rotation | **exposed** as `Rotation`; `Tiled.Sprite` applies it. A game reading the rectangle of a turned trigger must read `Rotation` too |
| object made from a template | **applied**: the template file (.tj) is read and the object's own values laid over it |
| text object | **exposed**: `Shape` is `TiledText`, the text in `Text` |
| property of a custom class | **exposed** as a nested `TiledProperties` |
| property pointing to an object | **exposed** as the object's id, an int, for `Tiled.Object(id)` |
| property pointing to a file | **exposed** as its path from the file system's root, not from the map's folder |

**Hidden means not drawn, and nothing else.** A hidden tile layer keeps its solid tiles, its place in paths and
its light shadows. That is how authors keep a `collision` layer: drawn in Tiled while editing, hidden for the game.

**Tall tiles.** A `TileMap` with `TallIn(parent)` stands its tall tiles (walls, pillars) as sprites in another
layer, placed when they are made, which is how Lantervale sorts walls with the hero. Hiding, tinting or moving the
`TileMap` does not reach them today. So this step also changes `TileMap`: its visibility and tint are carried onto
its tall sprites, and the layer's offset is set before they are made. Its tests include a hidden layer and an offset
layer with tall tiles in a `TallIn` layer.

**Templates** are the costly row of this step. A template's tile object names its own tileset,
so its tile has to be mapped into the map's tilesets, and added when the map does not use that tileset, which
touches decision 0067 (a map reads only the tilesets it uses). XML templates (.tx) are refused, as .tmx is.

## 3. Group and image layers

- **Group layers** are a tree, as in Tiled: a group is a `TiledLayer` of kind `TiledGroup` that holds its layers
  (`Layers()`, `Layer(name)`), with its own properties and settings. `Tiled.Walk` visits every layer in draw order
  for a game that does not care about groups.
- **A group's settings reach what it holds.** Each layer keeps its own values as the file writes them; what the
  engine applies to a `TileMap` or a picture is the folded value, as Tiled shows it: hidden when any group above is,
  opacity and tint multiplied, offsets added. `Shown()`, `FoldedOpacity()`, `FoldedTint()` and `FoldedOffset()` give
  a game the same folded values for the sprites it makes itself.
- **Image layers** load their picture as a `Sprite` at the layer's offset, in the map's points. `repeatx` and
  `repeaty` are refused until a game needs them.

## 4. Tiles

- **Animations keep each frame's duration.** `TileAnimation(id, fps, frames…)` stays for code-made tilesets; a Tiled
  tileset's animation plays by its own durations.
- **A tileset's drawing offset** (`tileoffset`) is applied; packs with tall tiles use it. Today it is dropped.
- **A tileset's object alignment** (`objectalignment`) is applied by `Tiled.Sprite`, as its anchor.
- **A tile's class** is exposed, beside its properties.
- **A tileset's render size and fill mode** (`tilerendersize`, `fillmode`, Tiled 1.9) are refused when they are not
  the default, until a map needs them.
- **Polygons and ellipses on a tile** stay bounding rectangles for now: `collide` has boxes and circles only, and a
  polygon solid is a change to `collide`, not to the loader. Until then the tileset tool shows the rectangle the
  engine uses, and the package doc says so. This is the one place the rule bends, and it is written down.

## 5. Declared classes

**Why classes need declaring.** In Tiled a class is a custom type of the project: a name, what it is used on (a
map, a layer, an object, a tile) and its properties with their defaults. A wall tile of class `wall` shows
`durability: 100` in Tiled without anyone typing it.

Tiled saves only the properties that were changed. The default lives in the `.tiled-project`, not in the map or the
tileset, so a wall left at 100 has no `durability` in the file at all, and a loader that does not know the class
reads nothing. The class has to be known at load for its defaults to exist in the game. This is why declaring
classes is not only a check: without it, class properties are simply wrong.

**Today.** Tile properties are declared in Go (`twod.TileProperties`, decision 0043): the loader checks them, and
`twod.TiledProject` writes them into the project file as one class, `ember.Tile`, with every property on every
tile. Maps, layers and objects have nothing: Lantervale's project file declares `level`, `exit`, `item` and the
rest by hand, and its Go code checks the same things by hand again. The two have already drifted: `teach`, `ambush`
and `slime_egg` are checked in Go and missing from the project file.

**Proposed.** One declaration, in Go, for every class:

```go
twod.Classes(
    twod.Class{Name: "wall", On: twod.OnTile, Properties: []twod.Property{
        {Name: "durability", Kind: twod.KindInt, Default: 100, Min: 1, Max: 1000},
    }},
    twod.Class{Name: "exit", On: twod.OnObject, Shape: twod.TiledRectangle, Properties: []twod.Property{
        {Name: "to", Kind: twod.KindString, Required: true},
    }},
    twod.Class{Name: "level", On: twod.OnMap, Properties: []twod.Property{{Name: "name", Kind: twod.KindString}}},
)
```

- **Defaults are filled.** A map, layer, object or tile of a declared class has every property of the class: the
  file's value where it was changed, else the default. `wall`'s durability reads 100.
- **Values are checked** as tile properties are today: a wrong kind, an enum value or a number out of range is an
  error naming the file, the layer, the object or tile, and the property. `Required` refuses a property left empty;
  `Shape` refuses an object of the wrong shape.
- **Unknown names are caught.** An object or a tile whose class is not declared, once any class is declared for its
  kind, is an error with the nearest name (`NearMiss` exists for properties). A property the class does not have is
  an error too. A game that wants loose maps declares nothing and gets today's behaviour.
- **The project file is written from the declarations.** `twod.TiledProject` writes every class with its use, its
  members and their defaults, so Tiled offers them typed. A test in the game fails when its `.tiled-project` is not
  what the declarations write, so the two cannot drift.
- **A property of a class type** (a class used as a member of another, Tiled's nested classes) is read as nested
  properties, defaults filled the same way.
- **`TileProperty` becomes `Property`**, since it now describes a property of anything. The engine's own tile
  properties (`solid`, `tall`, `shadow`…) stay on every tile, as the `ember.Tile` class does today; a tile's own
  class adds to them.

**Is it a good idea?** Yes, and it is close to required once classes are used at all, for the defaults. What it
costs and what to watch:

- **Go is the source, Tiled the editor.** A class added in Tiled's own type editor is overwritten the next time the
  project file is written, and refused at load. A designer adds a class by asking for it in Go (or in a data file
  the game turns into declarations). This is the same rule as tile properties today, and it is what keeps a map
  from carrying a property the game never reads.
- **One class per element.** Tiled gives an object or a tile a single class. A wall that is also a light is two
  properties of one class, or a class member, not two classes.
- **A changed default changes every map** that left the value alone, at the next load, with no edit to the maps.
  That is the point of a default, and it is also how a number changes without anyone seeing a diff in the level.
  For Lantervale, numbers the designer tunes belong in `data/*.yaml`, keyed by the class or the tile's name; a
  class default is for what belongs to the map (which door an exit leads to, how a wall is built).
- **The other way round** (Tiled's project file as the source, read at load) lets a designer add a class without
  Go, but nothing then checks that the game reads it, the project file must ship with the game, and it breaks with
  decision 0043. Not proposed.

A game's own rules (an item on walkable ground, an id that exists in its data) stay the game's.

## 6. Each system reads its own tile properties

**What is wrong today.** `twod` draws. Yet its Tiled loader knows six property names, and its `Tileset` stores what
they mean:

| Property or shape class | Whose it is | Today |
|---|---|---|
| `tall` | drawing: drawn from its foot, sorted by y | `twod`, rightly |
| `solid`, plain shapes, shapes of class `body` | collision and paths | named by `twod`'s loader, stored in `Tileset`, answered by `TileMap.SolidIn` and `Solid` |
| `shadow`, `occluder_group`, `height`, shapes of class `shadow` | light | named by `twod`'s loader, stored in `Tileset`, answered by `TileMap.ShadowsIn` |
| `emissive` | light | read and checked by the loader; no map tile glows from it |

A game with no lights still loads light properties, a new light property means editing the drawing package, and
`twod` declares a property (`emissive`) that nothing acts on.

**Proposed.**

- **The loader knows no names.** A tile keeps its properties (`Tileset.Value`) and its shapes, each with its class
  (`Tileset.Shapes(id) []TileShape{Rect, Class}`). What a name means is not the loader's business; it checks each
  against whatever was declared, as §5 says.
- **Each system declares and reads its own**, in a package that knows both sides:

| Package | Declares | Gives |
|---|---|---|
| `twod` | `tall` | as today |
| `collide/tiles` (new; imports `collide` and `twod`) | `solid`, the shape class `body` | `tiles.Of(tileMap)`: a `collide.Solid` and a `nav` grid over the map; `tiles.Solid(ids…)`, `tiles.Collision(id, shapes…)`, `tiles.BodyShapes(…)` for tilesets made in code |
| `twod/light` | `shadow`, `occluder_group`, `height`, `emissive`, the shape class `shadow` | the map's occluders, read by the light system itself; `light.TileHeight`, `light.TileOccluderGroup`, `light.TileShadow`, `light.TileShadowShapes` for tilesets made in code |

- **A game composes what it uses**, with its own:

```go
level, err := twod.LoadTiled(fsys, name, tiles.Properties(), light.TileProperties(), game.Classes())
walls, _ := level.TileLayer("walls")
solid := tiles.Of(walls) // a collide.Solid and a nav grid
space.AddSolid(solid)
paths := nav.New(solid)
```

- `collide` and `nav` stay free of `twod`, as today: `collide/tiles` is the one place that knows both.
- `TileMap.SolidIn`, `Solid`, `ShadowsIn` and `twod.ShadowCaster` go; so do `Tileset`'s solid, shadow, group, height
  and emissive fields and their options. `twod.Solid(ids…)` becomes `tiles.Solid(ids…)`.
- `emissive`: light either makes map tiles glow from it or stops declaring it. Chosen in the step, not left as it
  is.

**Four things the move must settle**, found by engine-render reading the code:

- **How light finds the maps.** Today light learns of a map's shadows by asking the collide space's solids whether
  they are a `twod.ShadowCaster`. Once the space holds `tiles.Of(map)`, light would see nothing. So light looks for
  tile maps itself: it keeps a view of the tile maps in the World, as it does of lights and occluders, and a map
  shown in a window casts there. A hidden map still casts, as decided.
- **`twod` draws with silhouettes.** Drawing reads a tile's shadow shapes in three places: a tall tile is lit at its
  base, lit tiles are drawn by their silhouette, and rooms use solids. `twod` cannot simply forget them. Light hands
  them over through the engine's internal lighting hook (`internal/lit`, which `twod` and light already share):
  `twod` asks "what is this tile's lit base" and draws with the answer, without knowing a property's name.
- **Endless maps.** `twod.InfiniteMap` is a solid and a shadow caster too, with its own `SolidIn` and `ShadowsIn`,
  and the horde's flow field reads it. It moves the same way, with the same tables, kept per chunk.
- **Paths.** `nav.New(tileMap)` reads `TileMap.Solid` today; it takes `tiles.Of(tileMap)` instead. That is a change
  in every game and example that finds paths, not a rename.

**Speed, measured first.** Solids and shadows are asked for every frame, so the move must not cost. Today's
numbers, on a temple-sized map (40×109 cells, a third of them walls, half of those with a collision shape; Apple M1
Max):

| Asked | Today | Why |
|---|---|---|
| the solids round one body (`SolidIn`, 48×48 points) | 148 ns, 2 allocations | three map lookups per cell, shapes placed and scaled each time |
| the shadows in one light's reach (`ShadowsIn`, 600×600 points, 361 cells) | 18.2 µs, 76 allocations | five map lookups per cell, a fresh slice per tile |
| is one cell solid (`Solid`, for paths) | 15 ns | three map lookups |

Today's typed fields are maps keyed by tile id, read cell by cell, so the current code is not the fast path it
looks like. Moving the data out is the occasion to build the fast one. Three layers of cache, each owned by the
system that reads it:

1. **Per tileset, a table by tile id.** Built once from the properties and shapes, the first time the system meets
   the tileset: a slice indexed by id holding what that system needs (collision: whole cell or its rectangles;
   light: casts or not, group, height, rectangles). One index instead of three to five map lookups. Rebuilt when
   the tileset changes, which a version counter on `Tileset` tells.
2. **Per map, the rectangles already placed.** For each cell, its rectangles in the map's space, flips and scale
   applied, in one flat slice with an offset per cell. A query walks the cells of its area and copies; nothing is
   placed, scaled or allocated per call (`AppendSolidIn(dst, area)` fills the caller's slice). Rebuilt when the
   map's `Version` changes; a single `TileMap.Set`, as a crumbling floor does, redoes that cell only.
The per-map table is keyed on the map's `Version` and its cell offset, and the tileset's version counter also
covers tilesets merged from several.

Merging runs of whole cells into one rectangle (a wall of twenty cells as one caster) is left out of this step. It
changes where casters join, so soft shadow edges can move by a pixel at the seams and the lit goldens could not stay
identical. It comes back, with a stated tolerance, only if the benches show light still needs it.

**The gate.** The step starts by adding these three benches to `ember/twod` on `wip`, with the numbers above as the
baseline, beside `nav`'s flow field bench, `bench2d` and Lantervale's frame bench. The step lands only when none is
slower at a fixed bench time, before and after on the same machine. The aim: `SolidIn` under
100 ns with no allocation, `ShadowsIn` under 5 µs with none, `Solid` under 5 ns. These are aims to check, not
promises; the table above is the only measurement so far.

**What it costs.** The light examples and panels, examples 27 to 29, the tileset tool, the guide's tile properties
and shadows pages and Lantervale's `levels`, `act` and `dungeon` all name the moved options and getters. The change
is wide, and for light and paths more than mechanical, which is why it is its own step, after G.

**Size.** engine-render's estimate is 8 to 9 days: the benches half a day, `collide/tiles` with the path grid a day
and a half, light declaring its properties, finding the maps and handing `twod` its silhouettes two days, the tables
for `TileMap` and `InfiniteMap` two days, the callers two days. That is more than steps A to D together.

## Still refused, on purpose

- **Infinite maps.** They would load as a `twod.InfiniteMap` of fixed chunks; no game asks for it.
- **Staggered and hexagonal maps**, **zstd**, **XML files**: save as JSON, as today.

## Order and owners

| Step | What | Owner | Tests |
|---|---|---|---|
| A | §1 the whole map, and the rule's test | engine-render (`ember/twod`) | class, size, layer order with tile and object layers mixed, object ids, duplicate names refused, `Tiled.Sprite` turned and resized against a golden map |
| B | §2 layer and object settings, tall tiles and templates included | engine-render | one map per row of the table: hidden, opacity, offset (each with tall tiles), rotation, text, each property kind; a template's tile object from a tileset the map uses, and from one it does not |
| C | §3 group and image layers | engine-render | a nested group: the tree, `Walk`'s order, its offset and opacity folded onto a `TileMap`; the same name in two groups; an image layer; `repeatx` refused |
| D | §4 tiles: per-frame durations, drawing offset, class | engine-render | an animation of uneven frames drawn at three times; a tileset with an offset |
| G | §5 declared classes | engine-render | a class default filled on a tile, an object, a layer and the map; each refusal; the project file written and read back by the loader |
| I | §6 each system reads its own tile properties: the benches first, then the move with the per-tileset and per-map tables, `InfiniteMap` included | engine-render (`ember/twod`, `twod/light`), engine-core (`collide/tiles`, `nav`) | the three benches and the frame benches before and after; every lit golden unchanged; a hidden map still casting; a tileset changed after first use; one cell set on a cached map; a map with a cell offset |
| E | Lantervale drops its second parse | lantervale-dev | `levels` reads class, size, layers and markers from `twod.Tiled`; `header`, `rawLayer`, `rawObject` are deleted; `offOf` uses `Tileset.ID`; after G, its markers are declared classes and its project file is written from them |
| F | The guide's Tiled page and example 29, updated with each step as it lands; the page after it, [Tile properties](../06-2d/12-tile-properties.md), becomes the "Classes and properties" reference of what the engine declares and reads, and at G of how a game declares its own | doc-writer | the guide test; a test that the section names every engine property and class |
| H | Lantervale's page of the classes and properties its maps may use (`lantervale/docs/04-world/`), kept current with each step | level-designer | after G, a test that the page names every class and property the game declares |

Each step is one branch and lands apart, in the order A, B, C, D, G, I. E follows A (and again G) and is a Lantervale
change, so it is shown to the user before it lands.

engine-render's rough sizes: A about 2 days, B about 3 days with templates, C about 1 day (it needs B's tall-tile
change), D about half a day, G 3 to 4 days, I 8 to 9 days.

**Cost at load.** Reading layers in order and their settings adds no pass over the cells. Step A is measured on
`temple.tmj` (40×109 cells, 111 tilesets) before and after; the load must not get slower.

## Decided (user, 2026-11-13)

1. `Layers()` returns every top layer as `[]*TiledLayer`; today's list becomes `TileLayers()`. We are the only
   users, so the clean name wins over unchanged callers.
2. Group layers are a tree.
3. A layer hidden in Tiled is hidden in the game.
4. Declared classes are built now (step G).
5. Hidden means not drawn: the layer still blocks bodies, paths and light.
6. Templates are built now, in step B.

7. Declared classes as §5 writes them: classes on maps, layers, objects and tiles, defaults filled at load, Go as
   the source.
8. Two references are kept current: the engine's classes and properties in the guide, on the page after Tiled maps (step F), the
   game's in Lantervale's docs (step H).

## Open

- §6 (each system reads its own tile properties, with its caches and its gate): the user's review. The user asked
  for the move, `solid` included, and for the speed to be checked and kept by caching (2026-11-13).

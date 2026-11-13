# Tiled maps

[Tiled](https://www.mapeditor.org) is a free level editor. A level drawn in it loads in one call: its layers come back in the file's order, tile layers as tile maps and object layers as objects that tell the game where to put things. Here a meadow with a pond and a little shrine, where the hero starts bottom-left and walls and pillars stop him.

![The hero walks up and right, to the shrine's wall, past a pillar](tiled.gif "ember · Tiled")

## The level file

`level.tmj` is the map as Tiled saves it in JSON: 20 × 12 cells of 32 points, two tilesets, `tiles.tsj` and `objects.tsj`, two tile layers, "ground" and "walls", and an object layer, "things", with the start and three pillars. In `tiles.tsj`, the wall tile has a `solid` property and the water an animation, so the map knows what stops the hero and what moves; [Tile properties](12-tile-properties.md) lists every property and class the engine reads. The pillars are tile objects, placed from `objects.tsj`, whose object alignment is `bottom`: an object stands on its bottom centre.

[level.tmj](../../examples/29-tiled/level.tmj)

## Loading it

`twod.LoadTiled` reads the map and its tilesets. `level.Layers()` gives its layers in the file's order, each a `*twod.TiledLayer` whose `Kind` says what it holds: `twod.TiledTiles`, a tile map from `TileMap()`, or `twod.TiledObjects`, objects from `Objects()`. The example walks them: each tile map goes in the window, with `space.AddSolid` so its solid tiles stop the hero, and the object layer spawns the hero and the pillars. `level.TileLayers()` and `level.TileLayer(name)` give the tile maps alone, `level.Layer(name)` a layer by name.

The cells are 32 points while the tiles are 16 pixels, so object positions are already in screen points. A layer may draw from several tilesets, a floor grid and a collection of wall pieces, for instance: the map's tilesets are merged, and each cell keeps its global id over them.

Only the tilesets the map places a tile from are loaded, pictures and all ([0067](../99-decisions/0067-a-tiled-map-reads-only-the-tilesets-it-uses.md)), so a level can carry a whole library of tilesets to pick pieces from in Tiled at no cost. `level.UsedTilesets()` lists those; `level.Tilesets()` lists them all, in order: an unused tileset of its own file isn't even read (its .tsj may be left out of the game) and is empty but for its `Source()`; an unused one embedded in the map keeps its properties and shapes, without pictures. A used tileset whose .tsj is missing is an error naming that file. `ImageFile(id)` names a tile's picture file. To draw a tile from a tileset the map never places, load that tileset with `twod.LoadTiledTileset`.

[main.go](../../examples/29-tiled/main.go)

## What the map says of itself

Nothing in the file is dropped without a word ([0068](../99-decisions/0068-a-tiled-map-hides-nothing.md)). The map tells its `Class()`, its size in cells, `Columns()` and `Rows()`, its cell size in pixels, `TilePixels()`, how many points a map pixel draws at, `Scale()`, and its `Background()` colour and `ParallaxOrigin()`.

A layer carries its `ID`, `Name`, `Class` and `Properties`, and its settings as Tiled saved them, and a tile layer's map draws as they say. A layer that isn't `Visible` is hidden, drawn no more, and still blocks bodies and casts shadows; its `Tint`, its alpha times its `Opacity`, tints its tiles; and the layer's `Offset` moves its tiles. Tall tiles stood up from the layer follow all three, and a tile map's `SetHidden` and `SetTint` reach them too. An object layer's `Offset` shifts its objects. `Parallax` and an object layer's `DrawOrder` are told, not applied. A tileset's drawing offset moves its tiles, on a map and in `level.Sprite`.

What the engine can't show yet is refused when the map loads, with an error naming the file, the layer or the object: image and group layers, two sibling layers of one name, infinite, staggered and hexagonal maps, `.tx` templates, a tileset's transparent colour, its class or its own properties, and a tile's class.

## The objects

The object layer's objects are the game's to read, and the game decides what each `Type` means: "start" is where the hero stands, and "pillar" a solid pillar. They go in a `SortByY` layer so the hero passes in front of and behind the pillars.

`level.Sprite(o)` stands a tile object as Tiled draws it: its tile's picture at the object's size, about the anchor its tileset's object alignment gives, turned by the object's rotation, and hidden when the object is. Its options are a sprite's, after those. The example adds a body to each pillar it returns.

An object keeps its `ID`, unique in the map, and `level.Object(id)` finds it again. Its `Rotation` is Tiled's, in radians, clockwise, and `Visible` is false for an object hidden in Tiled, which is listed all the same. Its `Rect` is Tiled's x, y, width and height, and its `Shape` says what was drawn: `twod.TiledRectangle` (also tile objects), `TiledEllipse`, `TiledPoint`, `TiledPolygon`, `TiledPolyline` or `TiledText`. A polygon's or polyline's `Points` are in the map's pixels, the object's position included. A tile object keeps its tile: `Tile` is nil for any other object. A text object keeps its `Text` and its `TextStyle`, how Tiled sets it; the engine draws no text, so the game draws it if it wants it. An object made from a template (`.tj`) comes back whole: the template's object, with the instance's own values over it.

```go
for _, o := range layer.Objects() {
	switch {
	case o.Tile != nil:
		things.Add(level.Sprite(o))
	case o.Shape == twod.TiledPolygon:
		addZone(o.Points)
	}
}
```

`Tile.ID` counts from 1 as a `Tileset` does, so Tiled's tile 5 is ID 6; `Tile.GID` is Tiled's own global id, flip bits included.

Custom properties come with their Tiled types. `level.Properties()` are the map's own, such as a room's name, tier or weight for a dungeon generator, a layer's are its `Properties` (or `level.LayerProperties("walls")`), and an object's are its `Properties`. A property that points to an object holds its id, a file property the file's path in the map's file system, and a property of a custom class its own `TiledProperties`, from `Class(name)`. Each reads by type, with whether it's there and of that type:

```go
name, _ := level.Properties().String("name")
tier, _ := level.Properties().Int("tier")
weight, ok := level.Properties().Float("weight")
```

Cells flipped or rotated in Tiled show as they look there. A tile map's `Flip(x, y)` says how a cell is turned, `twod.TileFlipX` across, `TileFlipY` upside down and `TileFlipDiagonal` for rotations, and `SetFlip` turns one; a tall tile takes the first two, not the diagonal.

[things.go](../../examples/29-tiled/things.go)

## As before

The hero and walking are those of [Collision](09-collision.md).

[hero.go](../../examples/29-tiled/hero.go)

[walk.go](../../examples/29-tiled/walk.go)

## Try it

[29-tiled](../../examples/29-tiled)

# Tiled maps

[Tiled](https://www.mapeditor.org) is a free level editor. A level drawn in it loads in one call: its tile layers become tile maps, and its objects tell the game where to put things. Here a meadow with a pond and a little shrine, where the hero starts bottom-left and walls and pillars stop him.

![The hero walks up and right, to the shrine's wall, past a pillar](tiled.gif "ember · Tiled")

## The level file

`level.tmj` is the map as Tiled saves it in JSON: 20 × 12 cells of 32 points, a tileset in `tiles.tsj`, two tile layers, "ground" and "walls", and an object layer, "things", with the start and three pillars. In the tileset, the wall tile has a `solid` property and the water an animation, so the map knows what stops the hero and what moves; [Tile properties](12-tile-properties.md) lists every property a tile can carry.

[level.tmj](../../examples/29-tiled/level.tmj)

## Loading it

`twod.LoadTiled` reads the map and its tileset. Each of `level.Layers()` is a tile map: the window shows it, and `space.AddSolid` makes its solid tiles stop the hero. The cells are 32 points while the tiles are 16 pixels, so object positions are already in screen points. A layer may draw from several tilesets, a floor grid and a collection of wall pieces, for instance: the map's tilesets are merged, and each cell keeps its global id over them.

Only the tilesets the map places a tile from are read and load their pictures ([0067](../99-decisions/0067-a-tiled-map-reads-only-the-tilesets-it-uses.md)), so a level can carry a whole library of tilesets to pick pieces from in Tiled at no cost. `level.UsedTilesets()` lists those; `level.Tilesets()` lists them all, in order: an unused tileset of its own file isn't even read (its .tsj may be left out of the game) and is empty but for its `Source()`; an unused one embedded in the map keeps its properties and shapes, without pictures. A used tileset whose .tsj is missing is an error naming that file. `ImageFile(id)` names a tile's picture file. To draw a tile from a tileset the map never places, load that tileset with `twod.LoadTiledTileset`.

[main.go](../../examples/29-tiled/main.go)

## The objects

`level.Objects("things")` lists the objects of that layer, and the game decides what each `Type` means: "start" is where the hero stands, and "pillar" is a solid sprite pivoted at its feet. They go in a `SortByY` layer so the hero passes in front of and behind the pillars.

An object's `Rect` is Tiled's x, y, width and height, and its `Shape` says what was drawn: `twod.TiledRectangle` (also tile objects and texts), `TiledEllipse`, `TiledPoint`, `TiledPolygon` or `TiledPolyline`. A polygon's or polyline's `Points` are in the map's pixels, the object's position included. A tile object, placed from a tileset, keeps its tile: `Tile` is nil for any other object. Its `Rect` stands with y at the picture's bottom, as in Tiled; Tiled's rotation isn't read.

```go
for _, o := range level.Objects("things") {
	switch {
	case o.Tile != nil:
		place(o.Tile.Tileset, o.Tile.ID, o.Tile.Flip, o.Rect)
	case o.Shape == twod.TiledPolygon:
		addZone(o.Points)
	}
}
```

`Tile.ID` counts from 1 as a `Tileset` does, so Tiled's tile 5 is ID 6; `Tile.GID` is Tiled's own global id, flip bits included.

Custom properties come with their Tiled types. `level.Properties()` are the map's own, such as a room's name, tier or weight for a dungeon generator, `level.LayerProperties("walls")` a layer's, and an object's are its `Properties`. Each reads by type, with whether it's there and of that type:

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

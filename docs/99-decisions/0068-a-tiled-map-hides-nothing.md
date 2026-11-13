# 0068. A Tiled map hides nothing

- Status: Accepted
- Date: 2026-11-13
- Area: assets
- Owner: user

## Context

`LoadTiled` dropped much of what a map holds without a word: its class and size, the order of its layers, an object's
id, rotation and visibility, a layer's settings. A game read its map twice to get them, and a map's author was never
told what the game ignored.

## Decision

- Everything a Tiled map can hold is applied, exposed on the API, or refused with an error naming the file, the layer
  and the object. What only the editor uses (terrain and Wang sets, export and editor settings, a layer's lock, a
  tile's probability) is listed as not read. A test fails on a key in none of these lists; a key a newer Tiled adds
  fails it until it is listed on purpose.
- `Tiled` says what the file says of itself (`Class`, `Columns`, `Rows`, `TilePixels`, `Scale`, `Background`,
  `ParallaxOrigin`) and keeps its layers in the file's order: `Layers()` returns `[]*TiledLayer` (tile, object, image
  and group layers, each with its settings) and `TileLayers()` the tile layers' maps. Objects keep their id
  (`Object(id)`), rotation and visibility; `Tiled.Sprite` stands a tile object as Tiled draws it, about the anchor its
  tileset's object alignment gives, at the object's own size.
- Group layers are a tree, as in Tiled; `Walk` visits every layer in draw order. Two layers of one name are refused
  among siblings.
- A layer hidden in Tiled is not drawn, and nothing else: it still blocks bodies, paths and light.
- Templates and declared classes (with their defaults) are read, built in later steps.

## Alternatives

- Keeping `Layers()` as the tile layers and adding `AllLayers()`: no caller changes, a worse name for good.
- Flattening groups into path-named layers: a game would never walk a tree, but a group's own settings and properties
  would have nowhere to live.
- A hidden layer ignored entirely, not solid either: a `collision` layer hidden for the game would stop working.
- Dropping silently what the engine doesn't do yet: the map looks right in Tiled and wrong in the game.

## Consequences

A game reads its map once, from `twod.Tiled`, and a map's author learns at load what the engine can't show. Each newer
Tiled field is a decision, made when a map first uses it.

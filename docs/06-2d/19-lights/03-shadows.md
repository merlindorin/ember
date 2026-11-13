# Shadows

![A lamp moving through the garden: its shadows shortening as it rises, then softening, then fading](shadows.gif "ember · shadows")

```go
set := twod.NewTileset(tex, cell, tiles.Solid(wall)) // solid tiles cast shadows
lamp := light.NewPoint(220, warm, light.Elevation(100), light.Size(12))
```

| Option | Default | |
|---|---|---|
| what casts: `tiles.Solid(ids…)` or collision shapes on a tile, `light.Occluder()` on a sprite | nothing | ![](thumb-occluders-on.png) |
| height of what casts | its picture | ![](thumb-height-8.png) |
| `light.Silhouette(groups…)`, on a sprite | its box | ![](thumb-silhouette-on.png) — fine for crowds: 1,000 under 64 lights cost about what their boxes do |
| `light.Elevation(points)` | 0, shadows to the radius | ![](thumb-elevation-100.png) |
| `light.Size(points)` | 0, sharp | ![](thumb-size-32.png) |
| `light.ShadowFade(curve)` | off | ![](thumb-fade-smooth.png) |
| `light.FadeFrom(share)` | 0 | ![](thumb-fade-from.png) |
| `light.ShadowColor(c)`, `light.ShadowOpacity(f)` | black, 1 | ![](thumb-shadow-color.png) |
| `light.BlockedBy(groups…)` | every group | ![](thumb-groups-blockedby.png) |
| `light.ShadedBy(groups…)`, on a lit layer or entity | every group | ![](thumb-shadedby-0-1.png) |

## Occluders

![The crates and columns not solid, casting nothing; then solid, the column and every crate casting](shadow-occluders.png "ember · shadows")

```go
set := twod.NewTileset(tex, cell, tiles.Solid(crate, column)) // or solid, or collision shapes, in Tiled
crate := twod.NewSprite(tex, light.Occluder())                // a sprite, over its whole box
camera := twod.NewCamera(light.OccludersFrom(space))          // a collide space's bodies and own solids
```

Light finds the tile maps of a window itself, hidden or not, and each casts there: a tile blocks light with its collision and its `shadow` shapes, or its whole cell when it's solid with no shapes, unless its `shadow` is `none` ([decision 0071](../../99-decisions/0071-each-system-reads-its-own-tile-properties.md)). A tile map in a collide space and in no window casts nothing. `light.OccludersFrom(space)` adds a space's hard bodies and the game's own solids, never its tile maps, which already cast. Changing a tile with `Set` costs light nothing until its next look at the map, which reads the map again whole: a `Set` then one light's reach on a 40 × 109 map, 81 µs.

```go
ruins := twod.NewTileMap(set, twod.CSV(cells), light.NoShadows()) // drawn only: its walls cast nothing
light.NoShadowsOf(ruins)                                          // true

seen := light.TileShadowsIn(nil, walls, area) // []light.TileShadowRect{Rect, Group, Height}
```

`light.NoShadows()` keeps a map from casting, whatever its tiles say: a map drawn as a picture only, whose walls something else stands for. `light.TileShadowsIn(dst, m, area)` appends what a map's tiles block light with in `area`, as light casts it, in the map's parent's space: for a game's rules that follow light, such as what blocks sight. It needs no window and reads the map as it is, `NoShadows` or not.

## Height

![The crate at height 24 and 8, under a lamp 100 points up: the lower crate's shadow ends sooner](shadow-height.png "ember · shadows")

```go
set := twod.NewTileset(tex, twod.Cell(16, 48), light.TileHeight(8, crate)) // in its pixels
box := twod.NewSprite(tex, light.Occluder(), light.Height(12))            // in its points
light.SetHeight(box, 20)                                                  // while the game runs
```

Unset, a tall tile stands as high as its picture, a solid one as its cell, others lie flat; in Tiled, the `height` [tile property](../12-classes-and-properties.md).

## Silhouette

![An orc as an Occluder casting a long wedge from its box; the same orc with Silhouette casting its own shape from its feet](silhouette.png "ember · shadows")

```go
hero := twod.NewSprite(sheet, twod.Animate(clips), light.Silhouette()) // its current frame, group 0
light.SetOccluder(hero)   // back to a box shadow, from the next frame
light.SetSilhouette(hero) // and its picture again; light.RemoveOccluder(hero) stops both
```

The picture lies from its foot, away from the light, as long as the sprite stands (its `light.Height`, else its picture): `d · h / (e − h)` under a light with an Elevation, `ShadowLength` under the sun. Every directional light and the three strongest lights reaching its foot cast it; static lights don't, and `Size` and `ShadowFade` don't change it yet ([decision 0046](../../99-decisions/0046-silhouettes-are-masks-subtracted-per-light.md)).

## Elevation

![Elevation 0, 100 and 200: every shadow to the radius, then the crate's ends, then the column's too](shadow-elevation.png "ember · shadows")

```go
lamp := light.NewPoint(220, warm, light.Elevation(100))
```

Something `h` tall, `d` from a light at elevation `e`, shadows the ground to `d · e / (e − h)`.

## Size

![Size 0 and 32: sharp shadows, then shadows softening away from what casts them](shadow-size.png "ember · shadows")

```go
lamp := light.NewPoint(220, warm, light.Elevation(100), light.Size(32))
```

## ShadowFade

![No fade, linear, smooth and soft, under a lamp at elevation 140](shadow-fade.png "ember · shadows")

```go
light.ShadowFade()                 // linear
light.ShadowFade(light.FadeSmooth) // or light.FadeSoft
lamp.SetShadowFade(light.FadeNone)
```

Linear is linear in light, so on screen it lifts fastest near the caster; smooth holds the dark longer.

## FadeFrom

![A linear fade, then the same keeping the shadow full over its first half](shadow-fade-from.png "ember · shadows")

```go
light.NewPoint(220, warm, light.ShadowFade(), light.FadeFrom(0.5))
```

## ShadowColor and ShadowOpacity

![The same shadows black, tinted blue, letting half the light through, and both](shadow-color.png "ember · shadows")

```go
lamp := light.NewPoint(220, warm, light.ShadowColor(color.RGB(0.15, 0.3, 1)), light.ShadowOpacity(0.5))
lamp.SetShadowOpacity(1)
```

In a light's shadow its light passes times `mix(1, ShadowColor, ShadowOpacity)`: every shadow of that light, boxes, tiles, silhouettes, a sun's, `ShadedBy` and static bakes, so two lamps' shadows can differ.

## Groups

![The crates in group 1: shadowed, then not under a light BlockedBy(0), while the column still shadows](occluder-groups.png "ember · shadows")

```go
set := twod.NewTileset(tex, cell, light.TileOccluderGroup(1, crate)) // or occluder_group in Tiled
box := twod.NewSprite(tex, light.Occluder(1))
lamp := light.NewPoint(220, warm, light.BlockedBy(0))
```

## ShadedBy

![A crowd whose feet boxes are in group 2: a monster's own box darkens its legs; with the crowd layer ShadedBy(0, 1), they're lit, while the column still shades the one behind it](shaded-by.png "ember · shadows")

```go
monster := twod.NewSprite(nil, twod.Size(16, 8), light.Occluder(2)) // a box at its feet
crowd := twod.NewLayer(1, light.Lit(), light.ShadedBy(0, 1))        // walls and props shade it, monsters don't
light.SetShadedBy(crowd, 0)
```

The mirror of `BlockedBy`: a light's `BlockedBy` picks what casts its shadows, a drawing's `ShadedBy` picks whose shadows fall on it and on what it holds. `ShadedBy()` with no groups takes none; static lights shade everything. Each set of groups in use draws the light picture once more a frame, up to three ([decision 0045](../../99-decisions/0045-shaded-by-draws-a-light-picture-per-set-of-groups.md)).

## In Tiled

| In the tileset | Blocks bodies | Blocks light |
|---|---|---|
| a collision object | yes | yes |
| a collision object of class `shadow` | no | yes |
| a collision object of class `body` | yes | no |
| `shadow` = `none` on the tile | yes | no |
| `occluder_group` = 0 to 2 on the tile | | in that group |

A tile casts its collision and its `shadow` shapes together, never its `body` shapes, and nothing at all with `shadow` = `none`; ellipses and polygons cast by their box. In code:

```go
tiles.Collision(id, rects…)        // bodies and light
light.TileShadowShapes(id, rects…) // light only
tiles.BodyShapes(id, rects…)       // bodies only: a rail, a bar's reach past its art
light.TileShadow(false, ids…)      // shadow = none
```

All the properties, and the system that declares each, are on [Classes and properties](../12-classes-and-properties.md).

![The tileset tool with wall tile 0 picked: its Inspector window a table of names and values, tall off with (default) after it, solid switched on, then the objectgroup with object 1 on one line, body · rectangle · 0,0 · 16×20, and object 2 unfolded, type shadow](tileset-tool.png "tileset · walls.tsj")

`go run ./tools/tileset path/to/tiles.tsj` edits a tile as Tiled stores it, in one table of names and values: its engine `properties`, then its `objectgroup`, the selected `object` unfolded with its `type` none, `shadow` or `body` (orange, blue, violet), its `shape` rectangle, ellipse or polygon, `x · y` and `width · height`, the others one line each. Its Preview, folded at first, lights the tile, its lamp's `shadows` switch off for a light through everything: hold the mouse in it to move the light, click it and walk the body with WASD, View › Static light (baked) bakes the lamp; View › Preview room…, or `-room path/to/room.tmj` on the command line, shows a room of the game with the tileset as edited. `-props props.json` adds the game's own tile properties to the table, and `-dir folder` sets where the open dialog starts until a tileset is opened.

![The tool previewing a crypt room: its walls and pillars outlined, the lamp lighting the floor, their shadows falling across it](tileset-tool-room.png "tileset · walls.tsj")

## All together

A lamp at the mouse over 64-point trees and pillars and 24-point crates. Keys: up/down or the wheel elevation, S size, F fade curve.

[main.go](../../../examples/37-shadows/main.go)

[lamp.go](../../../examples/37-shadows/lamp.go)

Unchanged from [36-point-lights](../../../examples/36-point-lights): garden.go.

## Try it

[37-shadows](../../../examples/37-shadows)

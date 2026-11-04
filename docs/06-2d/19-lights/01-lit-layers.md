# Lit layers and ambient

![The crypt built up: ambient only, then a lamp, then its shadows, then a crate drawn unlit](parts.png "ember · light")

```go
floor := twod.NewLayer(-1, light.Lit()).Add(ground)
night := twod.NewCamera(light.Ambient(color.RGB(0.05, 0.05, 0.06)))
win.Add(floor, light.NewPoint(140, warm, light.At(104, 150)), night)
```

| Option | Default | |
|---|---|---|
| `light.Lit()`, on a layer | unlit | ![](thumb-lit-on.png) |
| `light.Unlit()`, on a sprite or a tile map | lit | ![](thumb-unlit-on.png) |
| `light.Ambient(c)`, on a camera | black | ![](thumb-ambient-0.15.png) |
| `light.AmbientMap(pic)`, on a camera | none | ![](thumb-ambient-map.png) |
| `light.Exposure(f)`, on a camera | 1 | ![](thumb-exposure-1.6.png) |
| `light.PerPixel()`, on a tile map | tall tiles lit at their foot | ![](thumb-perpixel-on.png) |
| `light.Affects(layers…)`, on a light | every lit layer | ![](thumb-affects-floor.png) |

## Lit

![The floor's layer without and with Lit: drawn as it is, then lit by the lamp](lit-layer.png "ember · light")

```go
floor := twod.NewLayer(-1, light.Lit()).Add(ground)
```

## Unlit

![A crate in the column's shadow: dark when lit, as drawn with Unlit](lit-unlit.png "ember · light")

```go
crate := twod.NewSprite(tex, twod.At(72, 120), light.Unlit())
things := twod.NewTileMap(set, twod.TallIn(room), light.Unlit())
```

## Ambient

![Ambient black, dim and bright](lit-ambient.png "ember · light")

```go
camera := twod.NewCamera(light.Ambient(color.RGB(0.15, 0.15, 0.2)))
light.SetAmbient(camera, color.RGB(0.4, 0.4, 0.45))
```

## Exposure

![Exposure 0.6, 1 and 1.6: the room darker, as it is, then lifted away from the lamp](exposure.png "ember · light")

```go
camera := twod.NewCamera(light.Exposure(1.3)) // the player's brightness setting
light.SetExposure(camera, 0.8)
```

The light a lit drawing reads is multiplied by it before it's capped at full, so dim places lift and lit ones stay. Probes read the light unscaled, so it never changes what the game sees; unlit drawings ignore it.

## AmbientMap

![A black ambient, then an ambient map of moonlight fading from the left](lit-ambient-map.png "ember · light")

```go
moonlit := twod.NewTiledPicture(paint, twod.TexelSize(32, 32))
camera := twod.NewCamera(light.AmbientMap(moonlit))
```

How to paint one is on [Tiled pictures](../20-tiled-pictures.md#an-ambient-map).

## PerPixel and LitAt

![A lamp below the column: lit as a whole at its foot, then pixel by pixel with PerPixel](lit-perpixel.png "ember · light")

```go
things := twod.NewTileMap(set, twod.TallIn(room), light.PerPixel())
pillar := twod.NewSprite(tex, light.LitAt(0, 24)) // one light, taken at its base
```

A tall tile that blocks light takes the brightest light just past its sides; a flat wall tile takes it past its left, right and bottom sides, so it lights from the room, not from behind.

## Affects

![A lamp lighting every lit layer, then with Affects(floor) the floor alone, the crates and column left dark](affects.png "ember · light")

```go
floor, room := twod.NewLayer(-1, light.Lit()), twod.NewLayer(0, light.Lit())
glow := light.NewPoint(140, warm, light.Affects(floor)) // the floor and what it holds
glow.SetAffects()                                         // every lit layer again
```

A sprite follows its layer, and static lights light everything. Each set of lights left out costs one more light picture, shared with `ShadedBy`, up to four a window; past that, the extra layers take the window's own picture, lit by every light ([decision 0048](../../99-decisions/0048-a-light-affects-layers-through-a-picture-per-set-of-lights-left-out.md)).


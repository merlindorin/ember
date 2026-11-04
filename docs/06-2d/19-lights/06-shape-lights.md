# Shape lights and Emissive

![The hero walking past a glowing rune to a lava pool that warms the grass, a neon tube and a crystal glowing behind](shape.gif "ember · shape lights")

```go
neon := light.NewShape(light.Line(geom.V2(-32, 0), geom.V2(32, 0)), cyan, 40, light.At(96, 320))
lava := twod.NewSprite(tex, light.Emissive(80), light.Unlit())
```

| Option | Default | |
|---|---|---|
| shape | — | ![](thumb-shape-rect.png) |
| reach | — | ![](thumb-reach-80.png) |
| `light.Shadows(true)` | off for shape lights | ![](thumb-shadows-on.png) |
| `light.Emissive(reach)`, on a sprite | off | ![](thumb-emissive-on.png) |

## Shapes

![Rect, Circle, Capsule, Line and Polygon](shape-kinds.png "ember · shape lights")

```go
light.Rect(120, 40)
light.Circle(36)
light.Capsule(geom.V2(-50, -20), geom.V2(50, 20), 16)
light.Line(geom.V2(-70, 30), geom.V2(70, -30))
light.Polygon(geom.V2(0, -50), geom.V2(50, 36), geom.V2(-50, 36))
```

## Reach

![Reach 8, 24 and 80 round the same circle](shape-reach.png "ember · shape lights")

```go
light.NewShape(light.Circle(36), cyan, 80)
```

## Shadows

![A circle of light beside the column: casting no shadows by default, then the column's with Shadows(true)](shape-shadows.png "ember · shape lights")

```go
light.NewShape(light.Circle(10), cyan, 160, light.Shadows(true)) // takes 8 of the 256 shadow rows
```

## Emissive

![A flask drawn unlit, then glowing on the floor round it](shape-emissive.png "ember · shape lights")

```go
flask := twod.NewSprite(tex, light.Emissive(90), light.Unlit())
light.SetEmissive(flask, 0) // stops the glow
```

## All together

A lava pool, a neon line, a crystal circle with shadows and a polygon rune, and the hero walking past them unlit by any lamp. Keys: arrows walk, X the crystal's shadows.

[main.go](../../../examples/40-shape-lights/main.go)

[shape.go](../../../examples/40-shape-lights/shape.go)

[keys.go](../../../examples/40-shape-lights/keys.go)

Unchanged from [36-point-lights](../../../examples/36-point-lights): garden.go, hero.go, walk.go.

## Try it

[40-shape-lights](../../../examples/40-shape-lights)

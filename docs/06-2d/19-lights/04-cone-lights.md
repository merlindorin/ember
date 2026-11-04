# Cone lights

![A flashlight's cone sweeping round the hero, the trees and pillars throwing shadows out of it](cone.gif "ember · cone lights")

```go
beam := light.NewCone(220, warm, 0, 0.5, light.At(104, 150))
beam.SetAngle(toward.Angle()) // each frame
```

| Option | Default | |
|---|---|---|
| angle, in radians | — | ![](thumb-angle-up.png) |
| spread, each side | — | ![](thumb-spread-1.2.png) |
| `light.Size(points)` | 0, sharp | ![](thumb-cone-size-32.png) |

Radius, Falloff, Intensity, colour and Shadows work as on [Point lights](02-point-lights.md).

## Angle

![Angle 0, −π/2 and π/4: right, up, down-right](cone-angle.png "ember · cone lights")

```go
light.NewCone(220, warm, -math.Pi/2, 0.5)
```

## Spread

![Spread 0.2, 0.5 and 1.2 radians each side](cone-spread.png "ember · cone lights")

```go
light.NewCone(220, warm, -0.3, 1.2)
```

## Size

![Size 0 and 32: the column's shadow sharp, then softening away from it](cone-size.png "ember · cone lights")

```go
light.NewCone(220, warm, -0.3, 0.6, light.Size(32))
```

## All together

A flashlight in the hero's hand points at the mouse. Keys: arrows walk, F switches it, S its spread, K its size.

[main.go](../../../examples/38-cone-lights/main.go)

[cone.go](../../../examples/38-cone-lights/cone.go)

Unchanged from [36-point-lights](../../../examples/36-point-lights): garden.go, hero.go, walk.go.

## Try it

[38-cone-lights](../../../examples/38-cone-lights)

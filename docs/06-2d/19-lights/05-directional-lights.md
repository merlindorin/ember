# Directional lights

![A day over the garden: the shadows swing and shorten at noon, then night falls](directional.gif "ember · directional lights")

```go
sun := light.NewDirectional(color.RGB(1, 0.9, 0.75), 0.25*math.Pi)
win.Add(sun) // up to four a window
```

| Option | Default | |
|---|---|---|
| angle, where the light goes | — | ![](thumb-sun-angle-0.2.png) |
| `light.ShadowLength(points)` | 256 | ![](thumb-sun-length-120.png) |

## Angle

![Angle 0.2π, 0.5π and 0.8π: shadows to the lower right, straight down, lower left](sun-angle.png "ember · directional lights")

```go
sun.SetAngle(0.5 * math.Pi)
```

## ShadowLength

![Shadow length 40, 120 and 256](sun-length.png "ember · directional lights")

```go
sun.SetShadowLength(120)
```

## A day

![Dawn, noon and dusk: low warm light with long shadows, bright light with short ones, then low again](sun-day.png "ember · directional lights")

```go
high := float32(math.Sin(math.Pi * t)) // t from 0 at dawn to 1 at dusk
sun.SetAngle(math.Pi * (0.15 + 0.7*t))
sun.SetIntensity(high)
sun.SetShadowLength(60 + 260*(1-high))
light.SetAmbient(camera, moonlight.Lerp(daylight, high))
```

## All together

A day of 12 seconds, then 4 of night, looping. Keys: Space stops the clock.

[main.go](../../../examples/39-directional-lights/main.go)

[directional.go](../../../examples/39-directional-lights/directional.go)

Unchanged from [28-y-sort](../../../examples/28-y-sort): garden.go.

## Try it

[39-directional-lights](../../../examples/39-directional-lights)

# Point lights

![The hero walking between pillars at night, a flickering torch in his hand, the shadows swinging round him](point.gif "ember · point lights")

```go
torch := light.NewPoint(150, color.RGB(1, 0.75, 0.45), light.At(320, 192))
win.Add(twod.NewLayer(0, light.Lit()).Add(ground), torch, twod.NewCamera())
```

| Option | Default | |
|---|---|---|
| radius | — | ![](thumb-radius-220.png) |
| `light.Falloff(full)` | 0.25 | ![](thumb-falloff-0.5.png) |
| `light.Intensity(i)` | 1 | ![](thumb-intensity-2.png) |
| colour | — | ![](thumb-color-cold.png) |
| `light.Shadows(false)` | on, it casts shadows | ![](thumb-point-shadows-off.png) |

## Radius

![Radius 80, 140 and 220: the pool of light growing until the column and crates throw shadows](point-radius.png "ember · point lights")

```go
light.NewPoint(220, warm, light.At(x, y))
```

## Falloff

![Falloff 0, 0.5 and 1, each with its brightness curve from the lamp to the radius](point-falloff.png "ember · point lights")

```go
light.NewPoint(140, warm, light.Falloff(0.5))
```

The curve is the light's strength from the lamp (left) to the radius (right).

## Intensity

![Intensity 0.25, 1 and 2: faint, normal, bright](point-intensity.png "ember · point lights")

```go
light.NewPoint(140, warm, light.Intensity(2))
```

## Colour

![The same lamp warm, cold blue and green](point-color.png "ember · point lights")

```go
light.NewPoint(140, color.RGB(0.55, 0.75, 1))
```

## Shadows

![A lamp beside the column: casting shadows, the column's covers half the room; with Shadows(false), the light floods through](point-shadows-off.png "ember · point lights")

```go
light.NewPoint(180, warm, light.Shadows(false))
```

## At run time

```go
torch.SetRadius(200)
torch.SetIntensity(0.9)
torch.SetHidden(true) // puts it out
```

## All together

The hero carries a flickering torch through the garden at night; a lantern that casts no shadows glows by the gate. Keys: arrows walk, R the torch's radius, X its shadows.

[main.go](../../../examples/36-point-lights/main.go)

[point.go](../../../examples/36-point-lights/point.go)

[keys.go](../../../examples/36-point-lights/keys.go)

[hero.go](../../../examples/36-point-lights/hero.go)

Unchanged from [28-y-sort](../../../examples/28-y-sort): garden.go, walk.go.

## Try it

[36-point-lights](../../../examples/36-point-lights)

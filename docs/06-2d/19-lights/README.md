# Light

![The hero walking between pillars at night, a flickering torch in his hand, the shadows swinging round him](point.gif "ember · light")

![Ambient, plus a light, plus shadows, plus an unlit crate: the four parts of a lit picture](parts.png "ember · light")

```go
floor := twod.NewLayer(-1, light.Lit()).Add(ground)        // lit layers
space.AddSolid(walls)                                      // what casts shadows
camera := twod.NewCamera(
	light.Ambient(color.RGB(0.05, 0.05, 0.06)),            // the dark
	light.OccludersFrom(space),
)
torch := light.NewPoint(150, color.RGB(1, 0.75, 0.45), light.At(320, 192))
win.Add(floor, camera, torch)
```

| Page | |
|---|---|
| [Lit layers and ambient](01-lit-layers.md) | ![](thumb-lit-on.png) |
| [Point lights](02-point-lights.md) | ![](thumb-radius-140.png) |
| [Shadows](03-shadows.md) | ![](thumb-elevation-100.png) |
| [Cone lights](04-cone-lights.md) | ![](thumb-spread-0.5.png) |
| [Directional lights](05-directional-lights.md) | ![](thumb-day-noon.png) |
| [Shape lights and Emissive](06-shape-lights.md) | ![](thumb-shape-polygon.png) |
| [Probes](07-probes.md) | ![](thumb-probes.png) |
| [Debugging and performance](08-debugging.md) | ![](thumb-debug-occluders.png) |

Every option picture is one crypt, the lamp at the yellow dot, one call changed and printed under it.

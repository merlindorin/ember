# Probes

![The hero walking with his torch, violet slimes showing where the light reaches them and only their eyes in the dark](probe.gif "ember · probes")

```go
probe := light.NewProbe(twod.At(x, y))
if probe.Ready() && max(probe.Light().R, probe.Light().G, probe.Light().B) < 0.25 {
	hide(body)
}
```

| Method | |
|---|---|
| `Light()` | the light there, as a white lit pixel takes it, 1–3 frames late |
| `Ready()` | answered at least once |
| `Stale()` | out of view, or past 1,024 a window; keeps its last value |

Switch at two levels a little apart (here 25% and 27%), or a flickering torch makes a rule blink.

## All together

Each slime is a lit body, unlit eyes and a probe, side by side, since hiding an entity hides its children. Keys: arrows walk.

[main.go](../../../examples/41-probes/main.go)

[slime.go](../../../examples/41-probes/slime.go)

Unchanged from [36-point-lights](../../../examples/36-point-lights): garden.go, hero.go, point.go, walk.go.

## Try it

[41-probes](../../../examples/41-probes)

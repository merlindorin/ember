# Easing and paths

Easing shapes how a move speeds up and slows down; a path is the curve it follows. Here each family of easings slides a walker along its lane and back, side by side so their shapes compare, and another walker goes round a smooth loop.

![Eight walkers sliding along their lanes with different easings, and one going round a loop](easing.gif "ember · easing")

## The easings

`ease` has them by name: `Linear`, then `In`, `Out` and `InOut` of `Quad`, `Cubic`, `Sine`, `Expo`, `Back`, `Elastic` and `Bounce`. Each is an `ease.Func`, which turns a progress from 0 to 1 into an eased one. `In` starts slow, `Out` ends slow, and `InOut` does both; the lanes show the `InOut` of each family. `Back` and `Elastic` go past 0 and 1 on purpose, for an overshoot.

[easings.go](../../examples/35-easing/easings.go)

## Tweens

`ease.NewTween(2, fn, ease.Easing(…), ease.PingPong())` calls `fn` with the eased progress every frame for 2 seconds, then goes back. A tween is an entity, so it runs only once it is added to the World, here as a sprite's child, and leaves with the sprite; a tween nobody adds does nothing. `ember.New` runs tweens, and a bare `world.New` needs `ease.Plugin`. `fn` is where the game uses the progress, here `SetPosition` between two points with `Lerp`. `Delay` waits before starting, and `Loop()`, `PingPong()` and `Once()` repeat it as they repeat a sprite's clips. `Restart`, `Pause`, `Resume`, `Playing`, `Progress` and `SetEasing` control it while it runs, and a tween that doesn't repeat sends `ease.TweenFinished` when it ends, then leaves the World and its parent, so finished tweens don't pile up. What it moved stays: a fade-out that should take its entity away removes it on `TweenFinished`. To play a finished tween again, `Restart` it and add it again.

[main.go](../../examples/35-easing/main.go)

## A path

`geom.ClosedCatmullRom(loop, t)` is a point on a smooth closed curve through the points of `loop`, at `t` from 0 to 1. The curve is drawn as a line through 121 samples of it, and a tween with `Loop()` moves the walker round it. `geom` also has `CatmullRom` for an open curve through every point, and `QuadBezier` and `CubicBezier`; they work on both 2D and 3D points, with no path type of their own.

[path.go](../../examples/35-easing/path.go)

## Try it

[35-easing](../../examples/35-easing)

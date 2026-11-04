# A 2D camera

A level wider than the window, and a camera that follows the hero along it. The far hills scroll at half the speed of the ground, for depth, and a line of help stays put in the corner.

![The camera following the hero right, then left, with hills scrolling slower behind](follow.gif "ember · 2D camera")

## Layers and the camera

The hills on layer -1 are behind the ground on layer 0, and the help on layer 1 is in front. `Parallax(0.5)` moves the hills at half the camera's speed, and `Fixed` keeps the help where it is whatever the camera does. `twod.NewCamera` follows the hero, catches up with `Smoothing`, and never shows outside the level's `Bounds`: the level is as tall as the window, so the camera only scrolls sideways. The camera follows where the hero is drawn: when he moves in fixed steps, that is up to a step behind his `Position`, so a test that reads both sees a small gap.

[main.go](../../examples/23-2d-camera/main.go)

## The level

The ground is a polygon the whole level wide, and the hero stands on it. Parallax works both ways, so a layer at half speed also shows lower than the rest when the view is centred below its origin; the hills are drawn that much higher to sit on the horizon.

[level.go](../../examples/23-2d-camera/level.go)

## As before

The hero walks as in [Sprite sheets](02-sprite-sheets.md).

[walk.go](../../examples/23-2d-camera/walk.go)

## When the window resizes

This window has a fixed size. In one that can be resized, a camera by default shows more of the world as the window grows, at the same scale. `twod.ViewSize(w, h)` instead keeps at least that much world in view, scaling to fit: `ViewSize(0, 360)` always shows 360 points of height and lets the width follow. A camera with a `Resolution` scales by whole factors only, as on the [Pixel-perfect](13-pixel-perfect.md) page. A fixed layer's `Anchor(x, y)` puts its origin at that fraction of the window, so with `Anchor(1, 0)` a HUD at `At(-20, 20)` stays 20 points from the top-right corner, and a camera's `Layers(…)` makes it draw only those layers, as a minimap does.

## Nudging the view

`twod.Offset(x, y)`, or `SetOffset` while the game runs, moves what the camera shows by that many points. It applies last, after `Follow`, `Smoothing` and `Bounds`, so it may show past the bounds, and it never feeds back into the follow: setting it back to 0, 0 puts the view where following alone would have it. `ScreenToWorld` and `Center` include it. A camera shake is game code built on it: a system that sets a small random offset each frame for a moment, then 0, 0, for a moment when the hero is hurt.

## Try it

[23-2d-camera](../../examples/23-2d-camera)

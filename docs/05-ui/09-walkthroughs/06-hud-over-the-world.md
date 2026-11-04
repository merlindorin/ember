# HUD over the world

Three walkers wander a meadow, each with its name and a health bar that stay above it as it moves, while the score stays in the window's top-right corner. Click a walker to hurt it: its bar drops and the score counts the hit.

![Clicking Bo twice and Ada once, their bars dropping as they walk](hud.gif "ember · HUD over the world")

## Fixed and following

One Document draws over the meadow. The score is fixed to the window by its sheet: `position: fixed; right: 16px; top: 12px`, whatever moves under it. Each walker's tag, its name over its bar, is given `ui.Follow(sprite, geom.V2(0, -20))`: every frame, its bottom centre sits 20 points above the sprite on screen.

[main.go](../../../examples/16-hud-over-world/main.go)

## The walkers

Each walker is a sprite of the [sprite sheet](../../06-2d/02-sprite-sheets.md) and its tag. Its bar is a `kit.Progress` from 0 to 1 without its percentage, starting full.

[walker.go](../../../examples/16-hud-over-world/walker.go)

## Moving them

`Wander` walks each sprite back and forth along its route. It only moves the sprites; the tags follow on their own.

[wander.go](../../../examples/16-hud-over-world/wander.go)

## Hurting one

`Hit` reads the left click, finds the walker under the cursor, and lowers its bar with `kit.SetProgress`. Mouse positions are window points, the same as the sprites' here, since there's no camera.

[hit.go](../../../examples/16-hud-over-world/hit.go)

## As before

The meadow is the grass of the [tile map](../../06-2d/08-tile-maps.md).

[meadow.go](../../../examples/16-hud-over-world/meadow.go)

## Try it

[16-hud-over-world](../../../examples/16-hud-over-world)

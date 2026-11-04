# Y-sorting

In a top-down scene, what stands lower on the screen is nearer, so it draws in front. Here the hero walks up to a pillar and stops in front of it, then goes round and passes behind it, among trees and pillars standing on the grass.

![The hero stops in front of a pillar, then walks round and passes behind it](y-sort.gif "ember · y-sort")

## The garden

The grass is one tile map, and the trees and pillars another, from a picture in text like any level.

[garden.go](../../examples/28-y-sort/garden.go)

## Tall tiles

Trees and pillars are 16 × 32 tiles, taller than their cell. `Tall` marks them: they keep their shape and rise above their cell, standing at its bottom centre. `TallIn(garden)` puts them in the same layer as the hero, and `SortByY` on that layer draws everything in it by where its feet are, so the hero passes in front of and behind them. A tall tile sorts at the foot of its art, the bottom of its visible pixels, not of its cell, so a hero standing just below a column's base draws over it even when the picture has empty rows at its bottom; where it draws doesn't change. `Solid` makes their cells stop the hero, so he bumps into a trunk, not into the tree's leaves. A tile that isn't `Tall` draws whole too, never resized or cut: its bottom-left on its cell's, rising above the cell or spreading to its right, drawn with the map rather than sorted with the hero. So art meant to lie flat is cut to the cell's size in its sheet ([decision 0034](../99-decisions/0034-every-tile-draws-whole.md)).

[main.go](../../examples/28-y-sort/main.go)

## As before

The hero, pivoted at its feet with a body there, and walking, are those of [Collision](09-collision.md).

[hero.go](../../examples/28-y-sort/hero.go)

[walk.go](../../examples/28-y-sort/walk.go)

## Try it

[28-y-sort](../../examples/28-y-sort)

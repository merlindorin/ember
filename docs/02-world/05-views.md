# Views

A view is a group the World keeps in sync but never updates. Here a `Census` follows every burning entity, so the game ends when nothing burns, even with a rock left in the World.

## The view

`Census` is a `world.Group[Burning]`, with no `Update` method.

[census.go](../../examples/06-views/census.go)

## Using it

`AddView` registers it, and anything can read it: the quit function while the game runs, and `main` after it ends.

[main.go](../../examples/06-views/main.go)

## A rock

A rock has a `Name` but no `Heat`, so it is in the World but not in the `Census`.

[rock.go](../../examples/06-views/rock.go)

## As before

[cooling.go](../../examples/06-views/cooling.go)

[components.go](../../examples/06-views/components.go)

[flame.go](../../examples/06-views/flame.go)

## Try it

[06-views](../../examples/06-views)

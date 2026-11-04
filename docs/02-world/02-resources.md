# Resources

A resource is one value in the World, attached to no entity, that every system reads and writes. Here a `Weather` resource joins the flames of [Entities and systems](01-entities-and-systems.md): the wind rises, and the flames cool faster.

## The resource

`Weather` is a plain struct, and `Gusts` raises its wind every second through `world.Resource[Weather]`.

[weather.go](../../examples/03-resources/weather.go)

## Reading it

`Cooling` reads the same value to know how fast the flames cool.

[cooling.go](../../examples/03-resources/cooling.go)

## Adding it

`Insert` adds the resource to the World, before any system runs.

[main.go](../../examples/03-resources/main.go)

## As before

[components.go](../../examples/03-resources/components.go)

[flame.go](../../examples/03-resources/flame.go)

## Try it

[03-resources](../../examples/03-resources)

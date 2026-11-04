# Entities and systems

Components hold data, an entity is made of components, and a system updates every entity that has the components it needs. All game logic is written this way: here a candle and a campfire cool down until both are out.

## Components

A component is a plain struct, with a getter that lets a system ask for it.

[components.go](../../examples/02-entities-and-systems/components.go)

## An entity

An entity is a struct that embeds `*world.BasicEntity` and its components.

[flame.go](../../examples/02-entities-and-systems/flame.go)

## A system

`Burning` lists the getters the system needs, and `world.Group[Burning]` hands it every entity that has them. `Update` runs once per frame.

A system can have more than one group: the embedded one, and named fields such as `prey world.Group[Prey]`, which the World fills too. A group takes every entity that has the getters, whatever else it is. When two kinds of entity have the same components, say a hero and an enemy that both have health, a system that wants only one kind asks for a tag: an empty component that marks the kind, such as `type Hostile struct{}` with `func (h *Hostile) GetHostile() *Hostile`. Embed it in the enemy, add `GetHostile() *Hostile` to the system's interface, and the hero stays out.

A component's fields and getters sit on the entity as if they were its own, so a name that is already the entity's hides the other. A field called `Move` on a component hides the `Move` an embedded `twod.Item` gives the entity, and the entity then quietly leaves every group that needs that `Move`. Give fields and getters names of their own. When an entity is missing from a system, print `w.Explain(entity)`: it lists every system and view, whether the entity is in it, and what it lacks for the others, a missing getter or a method a field hides. Groups are matched when an entity is added: when a change after that makes it match a group, or stop matching one, `world.Regroup(entity)` offers it to its groups again, once the running system returns. `Explain` says when it's needed.

[cooling.go](../../examples/02-entities-and-systems/cooling.go)

## The World

`main` adds the entities and the system, and a function added with `On` stops the World once it is empty.

[main.go](../../examples/02-entities-and-systems/main.go)

## Try it

[02-entities-and-systems](../../examples/02-entities-and-systems)

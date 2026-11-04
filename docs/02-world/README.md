# The World

Everything in ember happens in one place: the **World**. It holds the entities, the systems that update them, the
resources they share and the presenters that draw them.

## Entities and components

An entity is a Go struct that embeds `*world.BasicEntity` and the components it is made of. A component is plain data
with a `GetX()` getter, so systems can ask for it through an interface.

## Systems

A system embeds `world.Group[T]`, where `T` is an interface listing the getters it needs. The World hands it every
entity that matches, as entities are added and removed. Small one-off logic can be a function instead, added with
`On(world.Update, name, fn)`.

## Phases

Each frame runs the systems phase by phase: `Startup` once, then `PreUpdate`, `FixedUpdate`, `Update` and `PostUpdate`.
A system runs in `Update` unless its `Phase()` method says otherwise. `FixedUpdate` follows simulated time instead of
frames, sixty steps per second, so the simulation is the same on a slow and a fast machine. `f.Time`, or `w.Time()`, holds the fixed
step, the interpolation alpha between two steps and the time scale, which pauses the game at 0 ([Time](07-time.md)).

## Resources

A resource is one value in the World that belongs to no entity, such as the weather or the game's settings.
`Insert(&Weather{})` adds it, and `world.Resource[Weather](w)` returns it to any system.

## Events

An event is a value one system sends and any other system reads, so neither has to know the other.
`world.Emit(w, WentOut{})` sends it, and a system holding a `world.Reader[WentOut]` reads each event once.

## Parents and children

An entity can own others: `parent.AppendChild(child)` makes `child` part of `parent`. Adding the parent to the World
adds its children, and removing it removes them.

## Views

A view is a `world.Group[T]` added with `AddView` instead of `AddSystems`. The World keeps it in sync with every entity
that matches, but never updates it, so any code can read which entities match `T`.

## Presenters

Drawing is not a system. After each frame is simulated, the World calls its presenters, added with `AddPresenter`, which
observe the entities they need through views. `ember.New` adds the renderer; a headless server has no presenter at
all. [Windows](../03-windows/README.md) shows one at work.

# world

The Entity Component System at the centre of ember. The `World` holds
entities, systems, views, resources and events, and runs them: phases, a
fixed step, the clock, plugins, presenters and the runner. It is pure Go, so
a dedicated server runs the same World as a windowed client.

```go
type Heat struct{ Degrees float32 }

func (h *Heat) GetHeat() *Heat { return h }

type Hot interface {
	world.BasicFace
	GetHeat() *Heat
}

type Flame struct {
	*world.BasicEntity
	Heat
}

type Cooling struct{ world.Group[Hot] }

func (s *Cooling) Phase() world.Phase { return world.FixedUpdate }

func (s *Cooling) Update(f world.Frame) {
	for e := range s.All() {
		if e.GetHeat().Degrees -= f.DT; e.GetHeat().Degrees <= 0 {
			f.World.RemoveEntity(e)
		}
	}
}

func main() {
	world.New().
		AddSystems(&Cooling{}).
		AddEntities(&Flame{world.NewBasic(), Heat{3}}).
		On(world.PostUpdate, "quit", func(f world.Frame) {
			if f.World.Len() == 0 {
				f.Quit()
			}
		}).
		MustRun()
}
```

[`examples/02-entities-and-systems`](../examples/02-entities-and-systems) starts with components, an entity and a system.

## Entities

An entity is a struct embedding `*world.BasicEntity` plus its components, each
with a `GetX()` getter, so systems ask for what they need through an
interface. `world.NewBasic()` and `world.NewBasics(n)` hand out IDs safely from
any goroutine.

`parent.AppendChild(child)` builds a hierarchy. Given the whole entity (not
only its `*BasicEntity`), adding the parent to a World adds the child too, as
removing the parent removes its descendants.

## Systems

A system has one method, `Update(f world.Frame)`. `Frame` carries `DT` (the
fixed delta in `FixedUpdate`), `Elapsed`, `FixedFrame`, the `World`, and
`Quit()`. Optional methods:

- `Phase()`: `Startup` runs once, then `PreUpdate`, `FixedUpdate` (as many
  fixed steps as the frame allows), `Update` and `PostUpdate`. `Update` by
  default.
- `Priority()`: higher runs first within a phase.
- `Name()`: the name timings show, the type name by default.
- `New(*world.World)`: called when the system is added.
- `Remove(world.Identifier)`: hears of each removed entity, for systems that
  keep entities of their own.

Embedding `world.Group[T]` routes every entity implementing `T` to the system,
including entities added before it. Every other `Group` field of the system,
named or not, exported or not, is filled the same way, so one system can
collect several kinds; `w.Explain(e)` lists each of them. `w.On(phase, name, fn)` adds a function
as a system.

## The World

Every method that sets the World up returns it, so a World reads as one chain.

| | |
|---|---|
| `AddSystems`, `AddEntities`, `On` | Simulation |
| `AddView(&group)` | A group kept in sync but never updated: presenters and replicators observe the World this way |
| `Insert(&r)`, `world.Resource[T](w)` | Resources, one per type; `Time` and `Stats` come with every World |
| `world.Emit(w, ev)`, `world.Reader[T]` | Events; each reader sees every event once |
| `AddPlugins` | `Plugin` (`Build(*World)`, built once per type) or `PluginFunc` |
| `AddPresenter` | Draws after each frame's systems, with the fixed-step alpha |
| `SetRunner`, `Run`, `MustRun` | The loop: `world.Headless` by default, the desktop window through `ember.New` |
| `Tick`, `Present`, `Frame`, `RunFrames(n)` | Drive frames by hand, as tests and servers do |
| `Quit`, `ShouldQuit` | Stop the runner after the current frame |
| `Stats()` | Smoothed per-system costs |

`RemoveEntity` called while a system runs is deferred until it returns.

# The World: entities, systems, phases, events, time

Package `github.com/lantervale/ember/world`. Guide: `docs/02-world`. Examples: `examples/02-entities-and-systems` … `examples/08-time`.

## Start a World

| Want | Call | Notes |
|---|---|---|
| A game with windows, drawing, sound | `ember.New(opts ...ember.Option) *world.World` | installs every engine plugin, see below |
| A game server: the same simulation, no GPU, builds with `CGO_ENABLED=0` | `server.New(opts ...server.Option) *world.World` | package `ember/server`; runner is `world.Headless` |
| A test of pure logic | `world.New() *World` | no plugins; runner is `world.Headless` |
| Run until something quits | `(w *World) MustRun()` / `Run() error` | `f.Quit()` or `w.Quit()` stops after the frame |
| Step by hand (tests) | `(w *World) RunFrames(n int)` | n frames at the fixed step (1/60 s) |
| One frame of a given length | `(w *World) Frame(dt float32)` | Tick + Present + event aging |
| Simulate only, no presenters | `(w *World) Tick(dt float32)` | |

`ember.New` adds: desktop windows (a default one only if the game adds none), the renderer, cameras' controllers, skeletal animation, the `ease` tween system, `audio`, the GPU and automation (`EMBER_AUTOMATION=host:port` to serve it).
Options: `ember.Headless()`, `ember.Automation(auto *automation.Automation)`, `ember.CrashLog(path string)`.
Without `ember.CrashLog`, `crash.log` and `errors.log` go in the user's config folder under the slug of the window's title; `EMBER_CRASH_DIR=dir` puts them in `dir/<slug>/` instead (tests and captures set it). A clean run leaves no folder it made itself.
`server.New` adds every one of those except the GPU and the desktop runner (decision 0064). Option: `server.Automation(auto *automation.Automation)`. Plugins are `AddPlugins(plugins ...Plugin)`, `Plugin` is `Build(w *World)`.

Builders all return `*World`, so chain them:

```go
world.New().
	Insert(&Weather{}).            // resources
	AddEntities(a, b).             // entities
	AddSystems(&Cooling{}, &Log{}). // systems
	On(world.Update, "quit", quitWhenEmpty).
	MustRun()
```

## Make an entity

An entity is a struct embedding `*world.BasicEntity` and its components. A component is plain data with a `GetX() *X` getter.

```go
type Heat struct{ Degrees float32 }

func (h *Heat) GetHeat() *Heat { return h }

type Flame struct {
	*world.BasicEntity
	Heat
}

f := &Flame{BasicEntity: world.NewBasic(), Heat: Heat{5}}
```

- `world.NewBasic() *BasicEntity` gives a fresh ID. `(e *BasicEntity) ID() uint64`.
- `(w *World) AddEntity(e Identifier)`, `AddEntities(...)`, `RemoveEntity(e Identifier)`, `Has(e)`, `Len()`, `Entity(id uint64) (Identifier, bool)`.
- Gotcha: a component **field** named like a getter method hides it, and the entity silently leaves the group. `w.Explain(e) string` says why it doesn't match.
- 2D entities (sprites, text…) come from `twod` and are already entities: see the 2D reference.

## Update entities every frame: a system

A system embeds `world.Group[T]`, where T is an interface of the getters it needs, plus `world.BasicFace`.

```go
type Burning interface {
	world.BasicFace
	GetHeat() *Heat
}

type Cooling struct{ world.Group[Burning] }

func (s *Cooling) Update(f world.Frame) {
	for flame := range s.All() {
		flame.GetHeat().Degrees -= f.DT
		if flame.GetHeat().Degrees <= 0 {
			f.World.RemoveEntity(flame) // safe inside the loop: applied after Update returns
		}
	}
}
```

- `System` is `Update(f world.Frame)`. Add with `AddSystem(s)` / `AddSystems(...)`.
- Group: `All() iter.Seq[T]`, `Len() int`, `Get(id uint64) (T, bool)`, `Slice() []T`.
- Every `world.Group` field of a system is filled, named or embedded, exported or not: one system can hold several groups.
- Groups take an entity when it's added. After a change that makes it match (or not), call `world.Regroup(e)`.
- Optional methods: `Phase() world.Phase`, `Priority() int` (higher runs sooner in its phase, default 0), `New(*world.World)` (called when added).
- One-off logic needs no type: `(w *World) On(phase Phase, name string, fn func(f Frame)) *World`.

## Phases and the frame

Order each frame: `world.Startup` (once, before the first frame), `world.PreUpdate` (sample input), `world.FixedUpdate` (0..n fixed steps), `world.Update` (default), `world.PostUpdate`.

`world.Frame` fields: `DT float32` (the frame's delta, or the fixed delta in FixedUpdate), `Elapsed float32`, `FixedFrame uint64`, `Time *Time`, `World *World`. `f.Quit()` stops the run.

- Simulation that must not depend on frame rate goes in FixedUpdate: `func (s *Physics) Phase() world.Phase { return world.FixedUpdate }`.
- Sample input in PreUpdate so FixedUpdate uses it the same frame.
- What FixedUpdate moves in 2D is drawn interpolated between steps automatically.

## Clock: pause, slow motion, interpolation

`(w *World) Time() *Time` (also `f.Time`). Fields:

| Field | Meaning |
|---|---|
| `Scale float32` | 1 normal, 0.5 half speed, **0 pauses** (fixed steps stop) |
| `Elapsed`, `Delta` | simulated, scaled |
| `RealElapsed` | unscaled, never paused: menus and animations during a pause |
| `FixedDelta` | 1/60 by default |
| `Alpha` | 0..1, how far the frame sits between two fixed steps: draw at `prev + (cur-prev)*Alpha` |
| `MaxDelta` | 0.25: clamps long frames |

```go
w.Time().Scale = 0 // pause
```

## Share one value: a resource

Game values only (score, settings, weather); the engine exposes none of its services as resources.

- Add: `(w *World) Insert(resources ...any) *World` with pointers: `Insert(&Weather{})`.
- Read/write: `world.Resource[T any](w *World) *T` (panics if missing), `world.TryResource[T](w) (*T, bool)`, `world.ResourceOrInit[T](w) *T`, `world.SetResource[T](w, v *T) *T`.

```go
world.Resource[Weather](f.World).Wind += 2 * f.DT
```

## Tell other systems something happened: events

- Send: `world.Emit[T any](w *World, ev T)`.
- Receive: a field `world.Reader[T]` in the system, then `(r *Reader[T]) Read(w *World) iter.Seq[T]`. `Any(w) bool` and `Count(w) int` consume like Read.
- Each reader sees each event once, then it ages out at the end of the frame.
- While the World has FixedUpdate systems, an event waits until a fixed step has begun after it, so every phase's readers see it once.
- Names are `<Subject><Past>`: `WentOut`, `ease.TweenFinished`, `audio.VoiceFinished`.

```go
type Log struct{ wentOut world.Reader[WentOut] }

func (s *Log) Update(f world.Frame) {
	for ev := range s.wentOut.Read(f.World) {
		fmt.Println(ev.Name, "went out")
	}
}
```

## Parents and children

- `(e *BasicEntity) AppendChild(child BasicFace)`: adding the parent to a World adds its children. Appending to a parent already in a World adds the child (and its subtree) now.
- Removing a parent removes its children.
- `RemoveChild(child *BasicEntity)` only detaches: the child stays in the World until removed.
- `Children() []BasicFace` / `Descendents()` return the entities appended (type-assert them). `Parent()`, `HasParent()`.
- `world.Of(e BasicFace) *World`: the World e is in, or nil.
- twod entities and windows use `Add(children...)` instead, which returns the parent.

## Watch entities without updating them: a view

A struct that embeds a `world.Group[T]` and has no `Update`, added with `(w *World) AddView(view)`. The World keeps it in sync. Renderers, minimaps and counters use them.

```go
type Census struct{ world.Group[Burning] }

census := &Census{}
w.AddView(census) // census.Len() is how many burn
```

## Move something over time: tweens (`ease`)

Needs `ember.New` (it installs the tween system). A tween is an entity: add it as a child of what it moves, or to the World.

- `ease.NewTween(seconds float32, apply func(t float32), opts ...ease.TweenOption) *ease.Tween`. apply gets the eased progress 0→1.
- Options: `ease.Easing(f ease.Func)` (default `ease.Linear`), `ease.Delay(seconds)`, `ease.Once()` (default), `ease.Loop()`, `ease.PingPong()`.
- Easings: `ease.InQuad`, `OutQuad`, `InOutQuad`, … `Cubic`, `Sine`, `Expo`, `Back`, `Elastic`, `Bounce`.
- A finished one-shot emits `ease.TweenFinished{Tween}`, then leaves its parent and the World.
- Methods: `Pause()`, `Resume()`, `Restart()`, `Playing() bool`, `Progress() float32`.

```go
sprite.Add(ease.NewTween(2, func(t float32) {
	sprite.SetPosition(from.Lerp(to, t))
}, ease.Easing(ease.InOutQuad), ease.PingPong()))
```

## Complete program: a headless World

`examples/04-events`, condensed: two flames cool down, a log hears each go out, and the run quits when the World is empty.

```go
package main

import (
	"fmt"

	"github.com/lantervale/ember/world"
)

type Heat struct{ Degrees float32 }

func (h *Heat) GetHeat() *Heat { return h }

type Name struct{ Value string }

func (n *Name) GetName() *Name { return n }

type Flame struct {
	*world.BasicEntity
	Heat
	Name
}

type Burning interface {
	world.BasicFace
	GetHeat() *Heat
	GetName() *Name
}

type WentOut struct{ Name string }

type Cooling struct{ world.Group[Burning] }

func (s *Cooling) Update(f world.Frame) {
	for flame := range s.All() {
		flame.GetHeat().Degrees -= f.DT
		if flame.GetHeat().Degrees <= 0 {
			world.Emit(f.World, WentOut{Name: flame.GetName().Value})
			f.World.RemoveEntity(flame)
		}
	}
}

type Log struct{ wentOut world.Reader[WentOut] }

func (s *Log) Update(f world.Frame) {
	for ev := range s.wentOut.Read(f.World) {
		fmt.Printf("%.1fs  %s went out\n", f.Elapsed, ev.Name)
	}
}

func main() {
	flame := func(name string, degrees float32) *Flame {
		return &Flame{BasicEntity: world.NewBasic(), Heat: Heat{degrees}, Name: Name{name}}
	}
	w := world.New().
		AddEntities(flame("candle", 1), flame("campfire", 2)).
		AddSystems(&Cooling{}, &Log{}).
		On(world.Update, "quit", func(f world.Frame) {
			if f.World.Len() == 0 {
				f.Quit()
			}
		})
	w.RunFrames(600)
}
```

## Complete program: a window and a tween

```go
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func main() {
	win := platform.NewWindow(platform.Title("tween"), platform.Size(640, 360), platform.FixedSize())
	label := twod.NewText("hello", twod.At(100, 180), twod.FontSize(32))
	from, to := geom.V2(100, 180), geom.V2(540, 180)
	label.Add(ease.NewTween(1.5, func(t float32) {
		label.SetPosition(from.Lerp(to, t))
	}, ease.Easing(ease.InOutQuad), ease.PingPong()))
	win.Add(label)
	ember.New().AddEntities(win).MustRun()
}
```

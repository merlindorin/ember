# Particles, easing and curves

Status: approved and built (`twod.NewParticles`, `ember/ease`). A design record: the guide documents the API as it is.

Two gaps from the raylib/Ebitengine comparison, approved after the 2D screen work: particles in twod, then easing, curves and tweens. Both follow the usual rules:
- 0007: options, no exported config;
- 0015: no World resources;
- 0017: the UI is 2D;
- 0018: At is the pivot, children go in with Add, Size is a box, Load/New, option and setter names pair, options are typed.

## 1. Particles

```go
fire := twod.NewParticles(spark,
	twod.Rate(40),
	twod.Lifetime(0.6, 1.2),
	twod.Velocity(40, 90),
	twod.Direction(-geom.Pi/2, 0.4),
	twod.Gravity(geom.V2(0, -30)),
	twod.ColorOverLife(color.Hex("#ffd27a"), color.RGBA(1, 0.2, 0, 0)),
	twod.SizeOverLife(6, 1),
	twod.Blend(twod.Add),
)
camp.Add(fire)
```

### The entity

`NewParticles(tex *Texture, …ParticlesOption) *Particles` is a 2D entity: placed, tinted, layered and ordered like a sprite, and drawn by the same batched canvas.
- With a texture, each particle shows it centred and fitted to the particle's size.
- With nil, each particle is a round dot.
- The name avoids Emitter, which audio will want.

### Options

Each option has a setter where a game changes it at runtime (`SetRate`, `SetDirection`, `SetColors`…).

| Option | What it does |
|---|---|
| `Rate(perSecond)` | continuous emission |
| `Burst(n)` | n particles at once when it starts |
| `OneShot()` | only the burst, then it is done |
| `Lifetime(min, max)` | seconds each particle lives, picked between the two |
| `Velocity(min, max)` | launch speed in points per second (`Speed` is taken by sprites) |
| `Direction(angle, spread)` | the cone it launches in: radians, with ±spread |
| `Gravity(x, y)` | acceleration in points per second² |
| `Drag(d)` | how much speed is lost per second, 0 to 1 |
| `Colors(start, end)` | colour over life, a two-point gradient |
| `Sizes(start, end)` | size over life: the side of each particle's box in points |
| `Spin(min, max)` | rotation speed in radians per second, picked per particle |
| `LocalSpace()` | particles move with the emitter; by default they stay where they were emitted, in its layer |
| `MaxParticles(n)` | a cap on live particles, 1000 by default; at the cap, new ones are skipped |
| `Seed(n)` | the random sequence; by default each emitter gets the next of a counter, so runs repeat |

- **Shared options:** `Blend(Add)` for fire, `Tint`, `Z` and `At` are the shared ones.
- **Methods:** `Emit(n)` adds a burst now, as for a click, and `SetEmitting(on)` pauses or resumes the rate.
- **Reading it back:** `Live()` is the count of live particles, and `Done()` reports whether a one-shot has no particles left.
- **Event:** `twod.ParticlesDone{Particles}` is emitted when a one-shot's last particle dies, so a game can remove it.

### Simulation

The simulation is CPU-side, in the twod system, before drawing, like sprite animation.
- **Storage:** particles live in a slice of plain values that is reused, so nothing is allocated per particle once warm.
- **Determinism:** the steps are deterministic for a given seed and the same frame times, so tests and GIF captures repeat.

## 2. Easing, curves and tweens

### Where it lives

- **Curves go in `geom`.** They are geometry on vectors and pure functions:
  - `QuadBezier(a, b, c, t)`;
  - `CubicBezier(a, b, c, d, t)`;
  - `CatmullRom(points, t)`: through every point, t from 0 to 1 over the whole path;
  - `ClosedCatmullRom(points, t)`: a loop back to the first point, with t wrapping.
  
  They are generic over `geom.Vec2` and `geom.Vec3`.
- **Easing and tweens go in a new package, `ember/ease`.** Easing is used by tweens, ui and games alike, and a tween is an entity with a system, which `geom` must not import.

### Easing

```go
y := ease.OutBounce(t) // t and y in 0..1
```

- **`ease.Func`** is `func(t float32) float32`.
- **The set:** `Linear`, plus `In`/`Out`/`InOut` of `Quad`, `Cubic`, `Sine`, `Expo`, `Back`, `Elastic` and `Bounce`: `ease.InQuad`, `ease.OutQuad`, `ease.InOutQuad`… as package values.

### Tweens

```go
slide := ease.NewTween(1.5, func(t float32) { box.SetPosition(from.Lerp(to, t)) },
	ease.Easing(ease.OutBack), ease.PingPong())
box.Add(slide)
```

- **The entity:** `NewTween(seconds, apply func(t float32), …TweenOption)` is an entity. Each frame the ease system advances it by `world.Frame.DT` and calls `apply` with the eased progress. It uses no goroutines and no resources.
- **Ownership:** added as a child of what it moves, it leaves with it.
- **Options:**
  - `Easing(f)` sets the easing (default Linear), paired with `SetEasing`;
  - `Delay(s)`;
  - `Loop()`;
  - `PingPong()`.
- **Methods:** `Restart`, `Pause`, `Resume`, `Playing`, `Progress`.
- **Event:** `ease.TweenFinished{Tween}` is emitted when a tween that doesn't repeat ends, read like `ui.Clicked`. The tween then leaves the World and its parent, so finished one-shots don't pile up; what it moved stays, for the game to remove on `TweenFinished` if it is done with it. To play it again, `Restart` it and add it again.
- **Installing:** `ember.New` installs `ease.Plugin`, as it does the camera controllers.
- **Moving along a curve:** a tween that follows a curve combines the two, `geom.CatmullRom(path, t)` or `geom.ClosedCatmullRom(loop, t)` inside `apply`, with no special type.

## Examples

| # | Example | Shows |
|---|---|---|
| 32 | `34-particles` | a campfire with Add sparks and Alpha smoke, and a one-shot burst where you click |
| 33 | `35-easing` | a row of sprites, one per easing, sliding with ping-pong tweens, and one following a Catmull-Rom path |

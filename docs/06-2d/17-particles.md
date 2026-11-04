# Particles

A particle emitter throws out many small pictures that move, change and fade on their own: smoke, fire, sparks. Here a campfire smokes, burns and sparks at night, and clicking throws a burst of embers that falls and is gone.

![A campfire with flames, smoke and sparks, and two bursts of embers where the window is clicked](particles.gif "ember · particles")

## The campfire

The fire is three emitters on one layer, around the logs. `twod.NewParticles` takes a texture, or `nil` for round dots, and options:
- `Rate` is how many particles a second;
- `Lifetime` and `Velocity` take a minimum and a maximum, and each particle picks its own between them;
- `Direction` is an angle and a spread, in radians, clockwise from the right, so `-geom.Pi/2` is up;
- `Gravity` pulls them, a vector in points a second per second, and `Spin` turns each one at a speed picked between two;
- `ColorOverLife` and `SizeOverLife` go from a start to an end over each particle's life, a size being the side of its box in points.

The smoke is drawn normally; the flames and the sparks `Blend(Add)`, so where they overlap they glow brighter. Particles stay where they were thrown even if their emitter moves, unless it has `LocalSpace()`. `MaxParticles` caps how many live at once, `Seed(n)` makes a run repeat exactly, and `Emit(n)` and `SetEmitting` throw or stop particles while the game runs.

[main.go](../../examples/34-particles/main.go)

## The puff

The smoke and the flames share a soft disc made in code, as in [Blend modes and masks](14-blend-modes.md).

[glow.go](../../examples/34-particles/glow.go)

## Bursts

A click adds an emitter where the mouse is with `Burst(80)` and `OneShot()`: it throws eighty embers at once and stops, and `Drag` slows them as they fall. When its last particle is gone, the World sends a `twod.ParticlesFinished` event, and `Bursts` removes the emitter.

[bursts.go](../../examples/34-particles/bursts.go)

## Try it

[34-particles](../../examples/34-particles)

# Time

The World's clock can run slower, stop, and tell a frame how far it sits between two fixed steps. Here a ball takes one step each fixed step: a second at normal speed, at half speed and paused, then a few frames at 144 fps drawn smoothly between steps.

## A ball on fixed steps

The ball moves one unit each `FixedUpdate` step and remembers where it was. `Drawn(alpha)` is where to draw it in a frame: between its last two positions, `alpha` of the way.

[ball.go](../../examples/08-time/ball.go)

## Scale and alpha

`w.Time()`, or `f.Time` in a system, is the clock. `Scale` multiplies simulated time: 0.5 is half speed and 0 pauses, so fixed steps stop and the ball stays put. At 144 fps a frame is shorter than a fixed step, so most frames take no step; `Alpha` says how far the frame is towards the next step, and drawing at `Drawn(Alpha)` makes the ball glide instead of stutter. `RealElapsed` is the real time the frames took, never scaled or paused (each frame still clamped by `MaxDelta`), for what keeps running while the game is paused, such as a menu's animations.

[main.go](../../examples/08-time/main.go)

## Try it

[08-time](../../examples/08-time)

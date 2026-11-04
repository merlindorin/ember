# Fixed step

`Update` runs once per frame, and `FixedUpdate` sixty times per simulated second, whatever the frame rate. Here a slow machine at 30 fps and a fast one at 120 fps each run one second.

## Two systems

`Steps` picks the `FixedUpdate` phase with a `Phase()` method, and `Frames` runs in `Update`, the default.

[counters.go](../../examples/07-fixed-step/counters.go)

## Running frames by hand

`w.Frame(dt)` runs one frame without a window or a clock, so both machines run exactly one second.

[main.go](../../examples/07-fixed-step/main.go)

## Try it

The frame counts differ, but both machines take sixty fixed steps.

[07-fixed-step](../../examples/07-fixed-step)

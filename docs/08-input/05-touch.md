# Touch

ember reads the fingers on a trackpad: where each one is, and when it lands, moves, rests and lifts. This page draws each finger on a picture of the trackpad.

![Two fingers on the trackpad, each labelled with its ID and phase, moving, resting, then lifting](touch.gif "ember · touch")

## Try it first

1. Use a Mac with a trackpad. Touch works on macOS only for now.
2. Run the example from the `ember` folder:

   ```sh
   go run ./examples/50-touch
   ```

3. Rest one or more fingers on the trackpad and move them. Each one shows as a spot with its ID and phase.

## A touch is a place on the trackpad

A trackpad touch is **not** a point in the window. Its `At` runs from 0 to 1 across the trackpad: (0, 0) is the pad's top-left corner and (1, 1) its bottom-right. The mouse cursor stays where it is; touching the pad doesn't move it to your finger. To draw a finger, the example maps that 0–1 place onto a drawn pad.

[pad.go](../../examples/50-touch/pad.go)

## Reading touches

`in.Touches()` lists the fingers down this frame. Each `input.Touch` has:

- `ID`: the same number for one finger, from the moment it lands until it lifts.
- `Source`: `input.Trackpad`.
- `At`: where it is, 0 to 1 on the pad.
- `Phase`: `input.TouchBegan` on the frame it lands, `TouchMoving` while it moves, `TouchResting` while it stays still, and `TouchLifted` on the one frame it leaves. The example prints them as `began`, `moving`, `resting` and `lifted`.

The World also gets events: `input.TouchStarted`, `input.TouchMoved` and `input.TouchEnded`, each with its touch. The example adds a spot on `TouchStarted`, moves it with `Touches()` and removes it on `TouchEnded`.

[fingers.go](../../examples/50-touch/fingers.go)

[main.go](../../examples/50-touch/main.go)

## Testing

A test puts fingers on the trackpad without one:

- `in.AddTouch(1, input.Trackpad, geom.V2(0.3, 0.5))` lands finger 1.
- `in.MoveTouch(1, geom.V2(0.6, 0.5))` moves it.
- `in.EndTouch(1)` lifts it.

The next frame sees them.

## What isn't supported

- **Windows and Linux.** There are no touches there yet; `Touches()` stays empty.
- **Touchscreens.** `input.Touchscreen` exists, with `At` in window points, but nothing produces those touches yet.

## Try it

[50-touch](../../examples/50-touch)

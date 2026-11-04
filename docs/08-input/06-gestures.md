# Gestures

ember turns two fingers on the trackpad into gestures: pinch, turn, pan and swipe. This page has a crate you zoom by pinching, turn with two fingers and move by sliding two fingers together.

![A crate pinched bigger, turned, moved by two fingers, then pinched smaller](gestures.gif "ember · gestures")

## Try it first

1. Use a Mac with a trackpad. Gestures work on macOS only for now.
2. Run the example from the `ember` folder:

   ```sh
   go run ./examples/51-gestures
   ```

3. Pinch to zoom the crate, turn two fingers to turn it, and slide two fingers to move it. Swipe, or press R, to put it back.

## Reading gestures

Each gesture is how much happened this frame, and 0 when nothing did:

- `in.Pinch()`: how much the fingers spread apart (positive) or closed (negative). Multiply a size by `1 + Pinch()`.
- `in.Rotation()`: how far the fingers turned, in radians, the same way as a twod rotation. Add it to a rotation.
- `in.Pan()`: how far two or more fingers moved together, in trackpad units, 0 to 1 across the pad. Scale it to the distance you want.
- `in.Swipe()`: the direction of a quick swipe, on the one frame it happens.

A two-finger slide also scrolls: it arrives as `Wheel()` too, so a page that scrolls with the wheel scrolls with the trackpad.

[handle.go](../../examples/51-gestures/handle.go)

[main.go](../../examples/51-gestures/main.go)

## Testing

A test makes gestures without a trackpad: `in.AddPinch(0.1)`, `in.AddRotation(0.2)` and `in.AddSwipe(geom.V2(-1, 0))`. A pan comes from two touches moved together with `AddTouch` and `MoveTouch`, as on the [Touch](05-touch.md) page. The next frame sees them.

## What isn't supported

- **Windows and Linux.** There are no gestures there yet; each one stays 0.

## Try it

[51-gestures](../../examples/51-gestures)

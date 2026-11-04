# 0026. Pads rumble through the platform's haptics

- Status: Accepted
- Date: 2026-10-30
- Area: core
- Owner: user

## Context

Games want pads to rumble, with a setting to turn it off. GLFW 3.3, which reads the pads, has no rumble, and the pure `input` package must stay free of cgo (0004).

## Decision

- `Gamepad.Rumble(low, high, seconds)` and `StopRumble()` in `input`, where a new call replaces the last one. Two strengths because the DualShock 4, the DualSense and Xbox pads have a heavy left motor and a light right one, and the system exposes each grip on its own. `seconds` is capped at 30, the longest continuous haptic event.
- `State.SetRumbleEnabled`/`RumbleEnabled` is one global switch. When it is off, every call is a no-op and the pads stop. `input.NewRumbleRecorder(in)` keeps the last rumble per pad for tests.
- The desktop runner hands the frame's rumbles to the platform after each tick. On macOS this is the GameController framework (`GCController.haptics` → `CHHapticEngine`, macOS 11 and later), with a GLFW pad matched to its controller by family (from vendor and product) and connection order. Windows and Linux do nothing for now, as with touch. A hidden window never rumbles.

## Alternatives

- SDL2 as a dependency: portable rumble, but a second input stack next to GLFW and a large C library to ship.
- hidapi with Sony's and Microsoft's output reports: every pad's protocol and every connection type (USB, Bluetooth) to write and keep up to date, and the system's own drivers in the way.
- A single strength: simpler, but it loses the low and high feel the pads have.

## Consequences

Rumble works on macOS with no new dependency, behind an API that a Windows (XInput) or Linux (evdev force feedback) backend can fill later. Matching by family and order could pick the wrong pad when two pads of the same kind connect in a different order in GLFW and in the system.

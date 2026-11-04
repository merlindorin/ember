# Gamepads

ember reads game controllers: PlayStation, Xbox, Nintendo and most others. This page shows how to read their buttons, sticks and triggers, and how to bind them like keys.

![Pad 0 pressing each face button, pulling its triggers and leaning its sticks, the hero walking, then pad 1 plugged in](gamepads.gif "ember · gamepads")

## Try it first

1. Plug in a pad, by USB or Bluetooth.
2. Run the example from the `ember` folder:

   ```sh
   go run ./examples/49-gamepads
   ```

3. Press buttons, pull the triggers and move the sticks. The drawn pad lights up as you do. The left stick and the cross walk the hero on the right. On macOS, ✕ (South) makes the pad rumble, and each trigger drives one of its motors.

The pad's name shows above it. A second pad shows under the first. If nothing shows, your controller isn't recognised as a pad: see [What isn't supported](#what-isnt-supported).

## Button names

Buttons are named by where they sit on the pad, not by what is printed on them. So `GamepadSouth` is the bottom face button on every pad:

| ember | PlayStation | Xbox | Nintendo |
|---|---|---|---|
| `GamepadSouth` | ✕ Cross | A | B |
| `GamepadEast` | ○ Circle | B | A |
| `GamepadWest` | □ Square | X | Y |
| `GamepadNorth` | △ Triangle | Y | X |
| `GamepadLeftShoulder` | L1 | LB | L |
| `GamepadRightShoulder` | R1 | RB | R |
| `GamepadLeftTrigger` | L2 | LT | ZL |
| `GamepadRightTrigger` | R2 | RT | ZR |
| `GamepadBack` | Share | View | − |
| `GamepadStart` | Options | Menu | + |
| `GamepadGuide` | PS | Xbox | Home |
| `GamepadTouchpad` | touchpad click (PS4, PS5) | — | — |
| `GamepadLeftStickPress` | L3 | LS | left stick press |
| `GamepadRightStickPress` | R3 | RS | right stick press |
| `GamepadUp`, `GamepadDown`, `GamepadLeft`, `GamepadRight` | the cross | the cross | the cross |

## Reading a button

A pad button reads like a key, from the window's input:

- `in.Down(input.GamepadSouth)` is true while the button is held.
- `in.Pressed(input.GamepadSouth)` is true on the one frame it goes down.
- `in.Released(input.GamepadSouth)` is true on the one frame it comes up.

These read every pad plugged in at once. `PadView` draws the pad from these reads every frame.

[pad.go](../../examples/49-gamepads/pad.go)

## Sticks and triggers

Sticks and triggers are not just on or off. `in.Value(b)` says how far they are pushed, from 0 to 1:

- `in.Value(input.GamepadRightTrigger)` is how far the right trigger is pulled.
- `in.Value(input.GamepadLeftStickLeft)` is how far the left stick leans left. Each stick has four of these: `…StickLeft`, `…StickRight`, `…StickUp` and `…StickDown`.
- `in.Stick(input.LeftStick)` gives the whole stick as a direction. Y is down, as on screen.

A trigger or a stick direction also counts as held once it is past half way, so `Down` and `Pressed` work on them too.

## The deadzone

A stick at rest is never exactly centred. So a small lean, under 0.2, reads as no lean at all. The lean past it is scaled back to 0–1. Change it with `in.SetDeadzone(0.1)`.

## Binding a pad in an action map

A pad binds in an [action map](03-actions.md) like any key. Here the arrows, the left stick and the cross all walk the hero:

```go
input.Bind(Left, input.KeyLeft, input.GamepadLeftStickLeft, input.GamepadLeft)
```

`Direction` keeps how far the stick leans: half way walks at half speed.

[walk.go](../../examples/49-gamepads/walk.go)

## Two players

`in.Gamepad(1)` is the pad in slot 1 only. Read it, or give it to the action map, for a second player:

```go
controls.Down(in.Gamepad(1), Left)
```

`in.Gamepads()` lists the pads plugged in. A pad also has `ID()`, `Name()` and `Connected()`.

## Plugging in and out

When a pad is plugged in, the World gets an `input.GamepadConnected` event, with its ID and name. When one is unplugged, it gets `input.GamepadDisconnected`. Read them like any event.

[plugs.go](../../examples/49-gamepads/plugs.go)

[main.go](../../examples/49-gamepads/main.go)

## Rumble

A pad has two motors: a heavy one on the left, a light one on the right. `pad.Rumble(low, high, seconds)` runs them at those strengths, from 0 to 1, for that long, up to `input.MaxRumbleSeconds` (30). A new call replaces the one playing, and zero strength or zero seconds stops it, as does `pad.StopRumble()`. ember has no ready-made patterns: a jolt, a heartbeat or an engine's hum is the game's own code, such as calling `Rumble` each frame with what it should feel now.

Here ✕ gives a 0.3 s jolt, and the triggers drive the motors live, the left trigger the heavy one and the right trigger the light one, so each can be felt alone.

[rumble.go](../../examples/49-gamepads/rumble.go)

Some players dislike it, so a game's settings usually have a Vibration switch: `in.SetRumbleEnabled(false)` stops every pad and makes every call do nothing until it's on again, and `in.RumbleEnabled()` reads it. It is on by default ([decision 0026](../99-decisions/0026-pads-rumble-through-platform-haptics.md)).

## Testing

A test plugs in a pad and moves it, with no real pad:

- `in.ConnectGamepad(0, "Wireless Controller")` and `in.DisconnectGamepad(0)`
- `in.SetGamepadButton(0, input.GamepadSouth, true)`
- `in.SetGamepadStick(0, input.LeftStick, geom.V2(0.6, 0))`
- `in.SetGamepadTrigger(0, input.GamepadRightTrigger, 1)`

The next frame sees them. `input.NewRumbleRecorder(in)` records the rumbles played: `rec.Last(id)` is the last one on a pad, `input.Rumble{Low, High, Seconds}`, the zero `Rumble` for a stop, and whether any was played; `rec.Reset()` forgets them.

[main_test.go](../../examples/49-gamepads/main_test.go)

## What isn't supported

- **Rumble outside macOS.** Rumble works on macOS 11 and later, through Apple's GameController (DualShock 4, DualSense, Xbox pads). Elsewhere, and in hidden windows (`EMBER_HIDDEN`), it does nothing, as with touch.
- **Pads on other windows.** Pads are read on the primary window's input only.
- **Controllers ember doesn't know.** ember recognises a controller from a list of known pads (GLFW's, from the SDL community). The common pads are on it, the DualShock 4 by USB or Bluetooth included. A controller that isn't on it doesn't show as a pad.

## Try it

[49-gamepads](../../examples/49-gamepads)

# Actions

An action is what the player means, such as walk left or plant, bound to the buttons that say it, which the player can change. Here the hero walks with WASD, the arrows or a pad's left stick, a ghost tree follows the pointer into the world, a click or Enter plants it there, and the wheel zooms.

![The hero walking, trees planted where the pointer clicks, the view zooming in and out](actions.gif "ember · actions")

## The action map

A game reads what the player means rather than which button they pressed. Its actions are values of its own type, and `input.NewActions` binds each to buttons: keys, mouse buttons and a pad's buttons alike, so planting answers the left button, Enter and the pad's south button. A pad's buttons are named by where they sit, `GamepadSouth`, `GamepadEast`, `GamepadWest` and `GamepadNorth`, so one binding means the same button on every brand of pad, and each way a stick leans is a button too, such as `GamepadLeftStickLeft`. `String` names an action when the bindings are saved.

[controls.go](../../examples/48-actions/controls.go)

## Walking

`Direction` reads four actions as one direction, with y down as on screen. Keys give a full step, and diagonals are no faster; a stick gives its lean, so half a push walks at half speed. The hero is the walker of [A 2D camera](../06-2d/05-a-2d-camera.md), with the sheet's rows facing up and down too.

[walk.go](../../examples/48-actions/walk.go)

## The pointer

`Mouse` is where the pointer is, in window points. The camera's `ScreenToWorld` turns it into the world under it, through the camera's follow and zoom, so the ghost tree stays under the pointer while the hero walks. `Pressed` on an action is true on the frame one of its buttons goes down. `Wheel` is in lines, up negative.

[garden.go](../../examples/48-actions/garden.go)

## Gamepads

A pad binds like any key, `input.Bind(PlantTree, input.MouseLeft, input.KeyEnter, input.GamepadSouth)`, and a stick's lean like four buttons. [Gamepads](04-gamepads.md) shows a pad's buttons, sticks and triggers, and a pad for each player.

A hint on screen should name what the player holds. `win.Input().LastDevice()` is the device used last: `input.KeyboardDevice` (the start), `MouseDevice`, `GamepadDevice` or `TouchDevice`. A key press or typed text, a mouse button, the wheel or a pointer move of more than 3 points, a pad button or a stick or trigger past halfway, or a new finger makes it that device, as each frame begins; a twitch of the mouse or a resting stick doesn't. So a menu's footer reads "Ⓐ Choose" after a pad press and "Enter Choose" after a key:

```go
hint.SetText("Enter Choose")
if win.Input().LastDevice() == input.GamepadDevice {
	hint.SetText("Ⓐ Choose")
}
```

## The window

The ghost tree is the pointer, so `SetCursor(platform.CursorHidden)` hides the system one; `CursorText`, `CursorHand` and the resize arrows `CursorResizeEW`, `CursorResizeNS` and `CursorResizeNWSE` are the others (the last shows as a crosshair for now), and `CursorArrow` hands the cursor back to the UI, which shows a hand over what clicks, a text cursor over inputs and the resize arrows over a resizable panel's edges. The bindings print as JSON, by the actions' names: `json.Unmarshal` reads them back into the same actions, and `controls.Bind(PlantTree, input.KeySpace)` replaces an action's buttons, for a rebinding menu, where `Bindings` lists an action's buttons and `Bound` finds the action a button already does.

[main.go](../../examples/48-actions/main.go)

## Try it

[48-actions](../../examples/48-actions)

# Input: keyboard and mouse

Status: a review of the input API; the proposals that were accepted are built (see the decisions). A design record: the guide documents the API as it is.

A review of `ember/input` as games use it (`win.Input()`), against:
- 0007 (options, no exported config);
- 0015 (no resources);
- 0018 (one way, one word);
- 0019 (options read back, `<Subject><Past>`, required data positional).

Touch waits; gamepads have their own section below. Nothing here changes the API yet: these are proposals for the architect to decide.

## What there is

`*input.State` is a window's input, read by systems through `win.Input()`. The backend fills it between frames.

| Area | Today |
|---|---|
| Keys | `Down(k)`, `Pressed(k)`, `Released(k)`; `Axis(neg, pos)` and `Vector(left, right, up, down)`; `Key` values are GLFW's, named after a US layout (`KeyA`, `KeyKP0`, `KeyGraveAccent`, `KeyWorld1`, `KeyLast`) |
| Mouse buttons | `MouseDown(b)`, `MousePressed(b)`, `MouseReleased(b)`, for `MouseLeft`, `MouseRight`, `MouseMiddle` |
| Pointer | `Mouse()`, documented as window pixels but really window points; `MouseDelta()` |
| Wheel | `Scroll()`, this frame's sum |
| Text | `Text() []rune`, this frame's typed characters |
| Drops | `Dropped() []string` |
| Clipboard and cursor | on the window, not the input: `ClipboardText()`, `SetClipboardText(s)`, `SetCursorVisible(on)` |
| Capture | two exported, writable fields: `MouseCaptured` (the UI holds the pointer) and `KeyboardCaptured` (a text field holds the keyboard) |
| Backend side | exported on the same type: `SetKey`, `SetMouse`, `SetMouseButton`, `AddScroll`, `AddText`, `AddDrop`, `AddRepeat`, `SetGamepad`, `BeginFrame`, `EndFrame`, and `RawDown`/`RawPressed`/`RawReleased`/`RawRepeated` for the UI |

It works, and the examples read it plainly. The rough edges are about one way per idea and what games see.

## Proposals, most important first

### 1. One trio for every button

Keys and mouse buttons have parallel names: `Down`/`Pressed`/`Released` and `MouseDown`/`MousePressed`/`MouseReleased`.

**Proposal:** one trio taking any `input.Button`, an interface that `Key` and `MouseButton` implement (and `GamepadButton` later).

```go
in.Pressed(input.KeySpace)
in.Pressed(input.MouseLeft)
in.Down(input.MouseRight)
```

The `Mouse…` button methods go. 0018: one word.

### 2. Games see reading, not feeding

The backend's methods (`BeginFrame`, `EndFrame`, `AddRepeat`, `SetGamepad`, the `Raw…` reads) sit beside the game's on `State`. The two capture flags are writable fields that any system can flip, which 0007 forbids.

**Proposal:**
- **Internal:** frame bookkeeping, repeats and the raw reads move to an internal feed (`ember/internal/inputfeed`), which the desktop backend and ui use.
- **Public, for tests and captures:** `SetKey`, `SetMouse`, `SetMouseButton`, `AddScroll`, `AddText` and `AddDrop` stay, documented as "simulate input"; the examples' tests and the capture scripts use them.
- **Capture:** read-only methods `PointerCaptured()` and `KeyboardCaptured()`, set by ui only.

### 3. The pointer, named once and in the right unit

- **The unit:** `Mouse()` returns window points, as GLFW's cursor callback gives them, not pixels as its doc says. That's the doc to fix: points are right, because the canvas, ui and `Camera.ScreenToWorld` all work in points.
- **Into the world:** a 2D game converts with the window's camera, `cam.ScreenToWorld(in.Mouse())`, which already accounts for zoom, follow and a `Resolution` camera's letterbox. Proposal: say so on the 2D Input page, and add nothing to `input`, which must not know cameras.
- **Naming:** keep `Mouse()` and `MouseDelta()`. "Pointer" would only pay off with touch, which waits.

### 4. Key names

- **Modifiers:** add `KeyShift`, `KeyControl`, `KeyAlt` and `KeySuper`, down when either side is. Every shortcut needs them, and games write `Down(KeyLeftShift) || Down(KeyRightShift)` today.
- **Keypad:** rename `KeyKP0`…`KeyKPEnter` to `KeyPad0`…`KeyPadEnter`. `KeyKP` is GLFW's abbreviation.
- **Sentinels:** hide `KeyLast` and `KeyWorld1`/`KeyWorld2`.
- **Layout:** document that a `Key` is a physical key position named after the US layout, so `KeyW` is the key AZERTY prints Z on, which is what movement wants. Characters come from `Text()`.
- **Mouse buttons:** add `MouseBack` and `MouseForward` (buttons 4 and 5).

### 5. Wheel, text, drops, clipboard, cursor

- **Wheel:** rename `Scroll()` to `Wheel()`. Scroll is what a Scroll element does with it; the input is the wheel, or a trackpad's two fingers, in lines.
- **Text:** `Text()` stays. IME composition (dead keys, CJK) is out of scope for now; a line on the page says so.
- **Drops:** `Dropped()` stays. A file dropped on a window is an input to that window.
- **Clipboard:** it stays on the window, but per 0019 it becomes `Clipboard()`/`SetClipboard(text)`.
- **Cursor:** `SetCursorVisible(on)` becomes a cursor shape, `win.SetCursor(platform.CursorArrow|CursorHidden|CursorText|CursorHand)` with `Cursor()`, so ui can show a text cursor over inputs and a hand over buttons. Locking the pointer for mouse look waits for the 3D pass.

### 6. Helpers

- **Axis:** `Axis(neg, pos)` stays.
- **Vector:** `Vector(left, right, up, down)` is a normalised direction, y down. Rename it `Direction`; "vector" says nothing.

## Should the engine have an action map?

Yes, a small one. Games should read intent ("jump", "move") rather than keys:
- **Rebinding:** players remap keys.
- **Two keys, one intent:** `KeyW` and `KeyUp` both mean up.
- **Gamepads:** a gamepad joins later without touching game code.
- **Automation:** `automation.Action` already exposes game intents to clients, and an action map is the input side of the same idea.

```go
type Act int

const (
	Jump Act = iota
	Left
	Right
)

controls := input.NewActions(
	input.Bind(Jump, input.KeySpace, input.KeyW),
	input.Bind(Left, input.KeyA, input.KeyLeft),
	input.Bind(Right, input.KeyD, input.KeyRight),
)

in := win.Input()
if controls.Pressed(in, Jump) { … }
dx := controls.Axis(in, Left, Right)
```

- **Typed actions:** `input.Actions[A comparable]` is a value the game builds and keeps, not a resource (0015). It is generic over the game's own action type, so a misspelt action fails to compile, as typed options do.
- **Reads:** `Down`/`Pressed`/`Released`/`Axis`/`Direction` take the window's `*State`. One map then serves every window, and tests feed a `State`.
- **Rebinding:** `Bind(action, buttons…)` doubles as the setter, `Bindings(action)` reads back (0019), and `Bound(button)` tells a rebinding menu what a key already does.
- **Saving:** bindings save as JSON, with keys by name (`"Space"`, `"W"`), for a settings file.
- **Not in it:** contexts and stacks (menu vs game), chords and hold durations. A game swaps maps for contexts; the rest waits until a game needs it.

### Alternatives

- **Leave it to games:** every 2D example already writes its own small map (`Walk` reads arrows), but rebinding and gamepads would be rewritten per game, and the guide could not teach one way.
- **String action names:** simpler to save, but typos are silent. Generic keys keep typing; JSON uses the names a game gives its actions.

## For the 2D Input page

Once decided, the page shows:
1. reading keys and buttons (`Down`/`Pressed`/`Released`);
2. moving with `Direction`, and the pointer in the world with `cam.ScreenToWorld(in.Mouse())`;
3. typed text and the clipboard;
4. an action map with rebinding.

The next free example number holds it.

## Decided

All seven proposals are approved and built.
- `input.Button` is sealed and implemented by `Key`, `MouseButton` and `GamepadButton`.
- The backend side is `ember/internal/inputfeed`.
- `SetKey`, `SetMouse`, `SetMouseButton`, `AddWheel`, `AddText` and `AddDrop` stay public, documented as simulating input for tests and captures.
- The window's cursor is `platform.Cursor` (`CursorArrow`, `CursorHidden`, `CursorText`, `CursorHand`), with `Cursor()`/`SetCursor`. The clipboard is `Clipboard()`/`SetClipboard`.
- The action map is `input.Actions[A comparable]`, with `NewActions`/`Bind`/`Bindings`/`Bound`/`Actions`, reads from a window's `*State`, and saves to JSON by name. `input.ParseButton(name)` looks a button up by the name it prints.
- The World begins every window's input frame before its first system runs, so `RunFrames` and `Tick` see what tests set with `SetKey`, `SetMouse`, `AddText` and `AddWheel`, from the next frame on, as a real event would. Games never import `inputfeed`.

## Gamepads

Controllers join the same API instead of a parallel one. Rumble waits: GLFW 3.3 has no rumble, and nothing portable does it without SDL.

- **Buttons:** a `GamepadButton` is a `Button`, so `Down`/`Pressed`/`Released` and the action map take it: `input.Bind(Jump, input.KeySpace, input.GamepadSouth)`. Names are positions, never a vendor's glyphs:
  - `GamepadSouth`, `GamepadEast`, `GamepadWest` and `GamepadNorth` for the face buttons;
  - `GamepadLeftShoulder`/`GamepadRightShoulder`;
  - `GamepadLeftTrigger`/`GamepadRightTrigger`;
  - `GamepadLeftStickPress`/`GamepadRightStickPress`;
  - `GamepadBack`, `GamepadStart`, `GamepadGuide`;
  - the pad's cross: `GamepadUp`, `GamepadRight`, `GamepadDown`, `GamepadLeft`.
- **Sticks and triggers:** they are analog buttons. Each stick direction is a button (`GamepadLeftStickLeft` … `GamepadRightStickDown`) whose value is how far the stick leans that way. A trigger's value is how far it is pulled.
  - `Value(button)` reads 0 to 1. Keys and mouse buttons read 0 or 1.
  - A button counts as down past half way.
  - `Axis` and `Direction` use the values, so one action map moves a hero by keys or by stick: `input.Bind(Left, input.KeyA, input.GamepadLeftStickLeft)`. `Direction` keeps an analog lean and caps the length at 1.
  - `Stick(input.LeftStick)` reads the whole stick as a vector.
  - **Alternative:** `BindStick(Move, LeftStick)` would make a second kind of binding. Directions as buttons keep one `Bind`, and rebinding and saving work unchanged.
- **Deadzone:** radial, per stick, rescaled so a slight lean starts slow. It defaults to 0.2, and `in.SetDeadzone(r)`/`in.Deadzone()` change and read it per window.
- **Several pads:** each connected pad has an ID, its slot, stable while it stays connected.
  - `in.Gamepads()` lists the connected ones, and `in.Gamepad(id)` is one, idle when absent.
  - A `*Gamepad` reads like the input: `Down`/`Pressed`/`Released`/`Value`/`Axis`/`Direction`/`Stick`, plus `ID()`, `Name()` and `Connected()`.
  - The input itself reads any pad, so a one-player game never picks one.
  - The action map reads from an `input.Reader`, either a window's `*State` or one `*Gamepad`, so each player can read their own pad through the same map.
  - The World emits `input.GamepadConnected{ID, Name}` and `input.GamepadDisconnected{ID}` when a frame begins.
  - Pads arrive on the primary window's input.
- **Simulation:** `ConnectGamepad(id, name)`, `DisconnectGamepad(id)`, `SetGamepadButton(id, button, down)`, `SetGamepadStick(id, stick, v)` and `SetGamepadTrigger(id, trigger, pull)`. They show from the next frame, like keys.
- **Removed:** the Xbox and PlayStation names, `SetGamepad` with its arrays, and `Gamepad`'s exported fields.

## Touch

GLFW 3.3 has no touch: on macOS it turns two fingers on a trackpad into wheel events, with no finger positions, pinch or rotation. So the desktop backend reads touches itself, and `input` gets a touch API.

- **Touches:** `in.Touches()` is this frame's touches. Each `input.Touch` has an `ID` (stable while the finger stays down), a `Source`, a position `At` and a `Phase`: `TouchBegan`, `TouchMoving`, `TouchResting` or `TouchLifted`.
  - A lifted touch is listed for the frame it lifts, then gone.
  - The World emits `input.TouchStarted`, `input.TouchMoved` and `input.TouchEnded` (each carrying the touch) as the frame begins.
- **Source, never pretended:**
  - On a `Trackpad`, `At` is where the finger is on the pad, from (0, 0) at its top-left to (1, 1) at its bottom-right, not a window point; the cursor is still the mouse.
  - On a `Touchscreen`, `At` is in window points.
  - A game that maps pad touches to the screen does so knowingly.
  - **Alternative:** the architect's sketch named the sources `Pad` and `Screen`. `Trackpad`/`Touchscreen` avoid confusion with gamepads.
- **Gestures**, this frame's, like the wheel:
  - `in.Pinch()` is the change of scale, 0 when none: multiply a zoom by `1 + Pinch()`.
  - `in.Rotation()` is the change of angle in radians, turning the way twod rotations do.
  - `in.Pan()` is how far two or more touches moved together, their mean movement, in their source's units.
  - `in.Swipe()` is a quick swipe's direction (-1, 0 or 1 on each axis), for page-like moves.
  - **Where they come from:** on macOS, pinch, rotation and swipe come from the system's gesture recogniser, so they match every other app. Pan is computed from the touches.
- **The wheel stays:** two-finger scrolls still arrive as `Wheel()`, so Scroll elements and zooms by wheel work as before. Pan is the same motion seen as touches.
- **Backends:**
  - **macOS:** the GLFW window's content view accepts indirect touches, and gains `touches…WithEvent:`, `magnifyWithEvent:`, `rotateWithEvent:` and `swipeWithEvent:`.
  - **Windows (WM_POINTER) and Linux (libinput):** they come later. There `Touches()` is empty and gestures read 0; nothing fails.
- **Simulation:**
  - `AddTouch(id, source, at)`, `MoveTouch(id, at)`, `EndTouch(id)`, `AddPinch(d)`, `AddRotation(d)` and `AddSwipe(dir)` show from the next frame, like keys.
  - `EMBER_CAPTURE_TOUCH` scripts them in captures.
  - A scripted run (EMBER_CAPTURE set, or EMBER_HIDDEN) reads no real keyboard, mouse, trackpad or gamepad, and rumbles nothing: only the EMBER_CAPTURE_* scripts reach the input (6c5faa6).

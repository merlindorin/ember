# Keyboard

The hero walks with WASD or the arrows and runs while Shift is held. Enter names him: what the player types goes into his name tag until Enter again, Backspace takes a letter off, and Ctrl or Cmd with V pastes.

![The hero walking, running, then being renamed Heron](keyboard.gif "ember · keyboard")

## Keys

`Down` says a key is held, `Pressed` that it went down this frame and `Released` that it came up; a press lasts one frame, however long the key is held. `Direction` reads four keys as one direction, normalised so diagonals are no faster, with y down as on screen, and `Axis` reads two as one number. A key is a place on the keyboard, named after the US layout: `KeyW` is the key AZERTY prints Z on, so WASD walks with the same fingers on any layout. `KeyShift`, `KeyControl`, `KeyAlt` and `KeySuper` match the key on either side, and `KeyLeftShift` or `KeyRightShift` one of them.

[walk.go](../../examples/47-keyboard/walk.go)

## Text

What the player types is text, not keys: `Text()` is the characters typed this frame, as the layout and the system make them, accents and capitals included, and repeated while a key is held. So a name is read from `Text()`, while Enter and Backspace, which type nothing, are read as keys. `win.Clipboard()` reads the clipboard's text, and `SetClipboard` writes it. Walking stops while the name is typed, or every W in it would walk the hero; a UI text input does the same for the game, and `KeyboardCaptured` says when one has the keyboard. A focused menu takes only the keys it moves by, which `Captured(key)` says, as on the [Controls](../05-ui/09-walkthroughs/03-controls.md) page.

[naming.go](../../examples/47-keyboard/naming.go)

## The window

The name tag is the hero's child, so it walks with him.

[main.go](../../examples/47-keyboard/main.go)

In a test, `SetKey` holds a key and `AddText` types a character, both seen by the next frame.

## Try it

[47-keyboard](../../examples/47-keyboard)

# Controls

A settings window with one of each control: a text input, a slider, two checks, a progress bar and a list that scrolls, in a section that folds. Each control tells the game what changed through its callback, and the line at the bottom shows the last one. The window drags by its bar, resizes by its edges and closes.

![Opening the save slots, clicking Fullscreen, dragging the volume down, scrolling the slots, dragging the settings by their title, widening them by their corner, closing them with the × and opening them again with Enter](controls.gif "ember · controls")

<!-- capture: EMBER_HIDDEN=1 EMBER_CAPTURE=docs/05-ui/09-walkthroughs/controls.gif EMBER_CAPTURE_FRAMES=200 EMBER_CAPTURE_KEYS="enter:355-358" EMBER_CAPTURE_MOUSE="320,200@0;320,244@15;320,244@25;left:18-21;290,210@40;290,210@52;left:44-47;398,150@65;398,150@78;left:78-112;300,150@108;300,150@115;320,307@125;320,307@160;wheel:3@135;wheel:3@150;320,41@170;320,41@182;left:182-215;240,31@210;240,31@218;428,383@235;428,383@245;left:245-275;520,418@270;520,418@280;65,39@290;65,39@320;left:300-310;300,250@335" go run ./examples/13-controls (the corner grab 2 points inside the grip: an edge point misses the window) -->

## The settings

`kit.Window("Settings", kit.Resizable(), …)` holds a `kit.Form` of named fields, a row of checks and a `kit.CollapsiblePanel` round a `kit.Scroll`. Each control takes its value and options: `kit.TextInput` its text and a `ui.Placeholder`, `kit.Slider` and `kit.Progress` a value in their `kit.Range`, the slider moving by `kit.Step`, `kit.Check` its text and `ui.Checked(true)`. The page's sheet places the window, caps its size, gives the list its height and pins the status line to the window's foot.

[main.go](../../../examples/13-controls/main.go)

## Their callbacks

Each control takes its reaction where it's made, in `newSettings`: one callback per kind of change.

| Control | Callback |
|---|---|
| check | `ui.OnChange` |
| slider, text input | `ui.OnInput` at each step or key |
| text input | `ui.OnSubmit` on Enter |
| the slots' section | `ui.OnToggle` |
| the window | `ui.OnMove`, `ui.OnResize`, `ui.OnClose` |
| Escape or the pad's East, nothing else taking it | `ui.OnBack` on the window: Escape or East from anything inside it |

The callbacks' bodies, in `status.go`, write the status line; `doc.Value(n)` and `doc.Checked(n)` read what a control holds, and the back one closes the settings. Each runs in the frame its change happens.

[status.go](../../../examples/13-controls/status.go)

## Setting a value from Go

`Loading` fills the progress bar in four seconds with `kit.SetProgress`, and starts again.

[loading.go](../../../examples/13-controls/loading.go)

## Keyboard and gamepad

The volume slider has the focus from the start (`autofocus`). Tab and the arrows or a pad's cross move it, Enter, Space or the pad's South press or toggle, left and right step the slider, and Escape or East closes the window ([Focus and keys](../05-focus-and-keys.md)).

![A pad steps the volume up, moves down to Fullscreen and turns it on, then left to Music and turns it off, then closes the settings with its east button; Enter opens them again](controls-pad.gif "ember · controls")

<!-- capture: EMBER_HIDDEN=1 EMBER_CAPTURE=docs/05-ui/09-walkthroughs/controls-pad.gif EMBER_CAPTURE_FRAMES=120 EMBER_CAPTURE_PAD="connect:0-240;right:20-23;down:50-53;south:80-83;left:110-113;south:140-143;east:170-173" EMBER_CAPTURE_KEYS="enter:205-208" go run ./examples/13-controls -->

## Closing and opening again

Closing only hides a window, so what was set in it stays. `Reopen` shows it again with `doc.Show` when Enter is pressed and it isn't open. The window's lights, dragging, resizing and folding are on the [Window](../07-kit/30-window.md) page; cards and tooltips on [Panel](../07-kit/08-panel.md) and [Tooltip](../07-kit/20-tooltip.md).

[reopen.go](../../../examples/13-controls/reopen.go)

## Try it

[13-controls](../../../examples/13-controls)

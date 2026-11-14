# A window

One window and what it can do: a title, a size and a background, size limits, fullscreen, and the things it shows. A label in its corner shows the window's size, and the ground is seen through an orbit camera.

![The window, with its size in the corner](window.png "ember · window")

## The window

A window is an entity: `platform.NewWindow` makes it from its options, and adding it to the World opens it. This one can't be resized below 320 × 180 (`MinSize`) and keeps a 16:9 shape (`AspectRatio`). A window presents its frames in step with the display's refresh, vsync, unless it's given `platform.VSync(false)`; then frames run as fast as they come, which suits measuring, but the window still shows at most one each refresh and draws the others without showing them, so the window system never stalls the game. The label, the ground and the camera are its children, so they show in it: the label is a `kit.Label` in a small ui Document, padded 16 points from the corner by its sheet, and `camera.NewOrbit` turns around the ground as you drag the mouse in the window.

[main.go](../../examples/09-window/main.go)

## Something to see

The ground is a prop: a transform and a model.

[ground.go](../../examples/09-window/ground.go)

## Reading it

`ShowSize` holds the window and its label, reads the window's size and sets the label's text every frame.

[size.go](../../examples/09-window/size.go)

## Fullscreen

Every option has a setter for while the game runs. `ToggleFullscreen` reads the window's own input and switches fullscreen with `SetFullscreen` when F is pressed.

[fullscreen.go](../../examples/09-window/fullscreen.go)

## When it fails

If the window can't open, or a panic escapes a frame, ember writes a `crash.log`: the reason and the panic's stack, the Go, GLFW and wgpu-native versions, and a "Graphics:" line with the backend (Metal, Direct3D 12, Vulkan or OpenGL) and the graphics card. It lands in the player's config folder, under the primary window's title made into a name, here for a window titled "Ember Quest": `~/Library/Application Support/ember-quest/crash.log` on macOS, `%AppData%\ember-quest\crash.log` on Windows, where a message box also gives the reason and the log's path, so a game built without a console never fails in silence. `ember.CrashLog(path)` writes it elsewhere. A machine with no graphics adapter WebGPU can draw with gets "Ember found no graphics adapter to draw with", naming what it tried, with advice on drivers, Remote Desktop and virtual machines; a game missing its WebGPU library gets "Ember couldn't load its graphics library, which ships beside the game. Reinstalling the game puts it back."

A test or a capture that runs a program which may fail keeps the log out of the player's folder with `EMBER_CRASH_DIR=dir`: `dir` stands in for the config folder, so the log lands in `dir/ember-quest/crash.log`. A path given to `ember.CrashLog` still wins. `golden.Run` sets it for each scene, to `crash` in the scene's test folder, unless the scene's own `Env` names another.

## Try it

[09-window](../../examples/09-window)

# Multiple windows

The primary window shows the scene, and a second window shows it from above. Each window has its own camera and its own label. Drag the mouse in the scene window to turn around the crates, and scroll to zoom: the map stays still.

![The scene in the primary window](scene.png "ember · scene") ![The map window, from above](map.png "ember · map")

## Two windows

Each window is made by `platform.NewWindow`: adding it opens it, and removing or closing it takes it away. The first window added is the primary one. Each window's camera and label are its children; the label is a `kit.Label` in a ui Document of its own. A camera or a UI element that is a window's child shows in that window and reads its input: the scene window's orbit camera follows that window's mouse, and the map window's fixed camera draws there. Closing a window takes its camera and label with it. A camera outside any window can still pick one with `In(win)`.

[window.go](../../examples/10-multi-window/window.go)

## The World

`main` adds the scene window first, so it is the primary one, then the ground, the crates and the map window, like any other entities.

[main.go](../../examples/10-multi-window/main.go)

[scene.go](../../examples/10-multi-window/scene.go)

## Try it

[10-multi-window](../../examples/10-multi-window)

# Windows

A window is an entity like any other, made with `platform.NewWindow`: add it to the World to open it, and remove it, or close it, to take it away with its children. The first window added is the primary one, and `ember.New` opens a default window only when the game adds none. Each window has its own cameras and its own UI, and shows the camera with the highest `Priority` among those that target it: a camera or a UI element that is the window's child shows in that window, and reads its input, and one that belongs to no window shows in the primary one, unless a camera names its window with `In`.

The windows are where the renderer, one of the World's [presenters](../02-world/README.md#presenters), draws; a headless server opens none.

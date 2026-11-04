# Input

Each window has its input, `win.Input()`: what its keys, mouse buttons, pointer, wheel, gamepads and trackpad did this frame, read by the systems that need it. A window's input is a snapshot taken before the frame's first system runs, so every system sees the same keys down, and a press lasts exactly one frame. Input that arrives between frames, from the player or from a test, shows in the next one.

The pages build on the 2D section's sprites and cameras: the keyboard first, then the mouse, then actions, which let a game read what the player means whatever key, button or pad says it, and let the player rebind it, then gamepads, and last the trackpad's touches and gestures.

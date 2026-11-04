# Mouse

Three crates to drag with the left button, a wheel that zooms, and the mouse's Back and Forward buttons to undo and redo the moves. The pointer becomes a hand over a crate, and a picture of the game's own while one is carried.

![Two crates dragged, the moves undone and redone, the view zooming in and out](mouse.gif "ember · mouse")

## The pointer and the world

`Mouse()` is where the pointer is, in window points from the top-left corner. The world under it depends on the camera, so `ScreenToWorld` turns it into world units, through the camera's position and zoom: at zoom 2, a hundred points across the window are fifty in the world. `MouseDelta` is how far the pointer moved this frame. The buttons are read like keys, `Down`, `Pressed` and `Released`, with `MouseLeft`, `MouseRight`, `MouseMiddle`, and `MouseBack` and `MouseForward` for the side buttons. `Wheel` is this frame's turn of the wheel, a trackpad's two-finger scroll too, in lines, up negative. `SetCursor` gives the pointer a shape, `CursorHand` over what can be dragged and `CursorArrow` elsewhere, which also hands the cursor back to the UI: it shows a hand over buttons and a text cursor over inputs by itself. `CursorHidden` hides it, for a game that draws its own.

[drag.go](../../examples/47-mouse/drag.go)

## The window

A camera that follows nothing looks at the world's origin, so the crates are placed around it.

## A cursor of your own

![The move cursor: four arrows from a centre](move-cursor.png)

`platform.NewCursor(img, hotX, hotY, opts…)` makes a cursor from a picture, one pixel a point, its hot spot in the picture's pixels from the top-left; `win.SetCursor` takes it like `CursorHand`. `platform.HiDPI(img2x)` gives the same picture at twice the size for 2× screens on the web, Windows and Linux; macOS scales the plain one. Make each cursor once, like a texture. While a game's cursor is set, the UI's hand and text cursors stand aside, and `CursorArrow` gives them back ([decision 0061](../99-decisions/0061-picture-cursors-are-cursor-values.md)). `moveCursor` reads the example's 24-pixel move cursor and its 48-pixel double, centred on 12, 12.

[main.go](../../examples/47-mouse/main.go)

When the UI holds the pointer, over a button for instance, `PointerCaptured` says so and gameplay can leave that click alone. `Dropped` lists the files dropped on the window this frame. In a test, `SetMouse`, `SetMouseButton`, `AddWheel` and `AddDrop` play the player.

## Try it

[47-mouse](../../examples/47-mouse)

# Clipping

A layer can show its items only inside a rectangle, like a window into it. Here the same news ticker scrolls twice: freely across the top of the window, and clipped to a framed strip below, where it slides in and out of view.

![A ticker scrolling freely above, and the same ticker clipped to a framed strip](clipping.gif "ember · clipping")

## The program

`twod.NewLayer(1, twod.ClipTo(strip))` draws its items only inside `strip`, a rectangle in the layer's space, here the window's points. Each ticker is a text placed by its left edge with `Pivot(0, 0.5)`; the blue frame is a closed line around the strip, drawn on its own so it isn't clipped.

[main.go](../../examples/25-clipping/main.go)

## Scrolling

`Scroll` moves both texts left every frame and starts each one again from the right once its whole width, from `Measure`, has gone by.

[scroll.go](../../examples/25-clipping/scroll.go)

## Try it

[25-clipping](../../examples/25-clipping)

# 0021. A 2D view resizes by its camera

- Status: Accepted
- Date: 2026-10-30
- Area: rendering
- Owner: architect

## Context

Games are resized, maximized and made fullscreen, and each wants its world to answer differently: an editor shows more of it, an action game keeps the same framing, pixel art keeps whole pixels.

## Decision

- The camera decides, with options that read back and have setters:
  - by default, a bigger window shows more world at the same scale;
  - `ViewSize(w, h)` keeps at least that much world in view and scales to fit, 0 freeing an axis;
  - `Resolution(w, h)` draws at a fixed size scaled by whole factors and letterboxed, or by any factor with `FitWindow()`.
- Fixed layers and the UI stay in window points and hold to an edge: `Anchor(x, y)` on a fixed layer, `ui.Anchor` on elements.
- Everything is laid out again in the frame the window changes. The desktop draws frames while the OS holds the loop in a live resize, and a window leaving fullscreen goes back to where and how it was, maximized included.

## Alternatives

- **A window-level stretch mode** that scales everything, UI included: one switch, but a HUD would grow blurry with the world, and two cameras in one window couldn't differ.
- **One enum, `ResizeMode(Expand|KeepHeight|KeepWidth|Integer)`:** a single word to learn, but it says nothing of the size to keep, needs a second option for it, and duplicates `Resolution`'s scaling.
- **Leave it to games**, by setting Zoom from the window size each frame: no API, but every game repeats it and gets a frame late after a resize.

## Consequences

A camera's options describe its whole answer to resizing, and the HUD never scales with the world. A game wanting the UI to scale too sets the UI's scale itself.

# 0061. Picture cursors are Cursor values

- Status: Accepted
- Date: 2026-11-03
- Area: api
- Owner: architect

## Context

Games want cursors of their own (a bow's attack cursor in the demo), but a window's cursor was one of a few standard shapes, an `int` the UI and `SetCursor` share.

## Decision

- `platform.NewCursor(img, hotX, hotY, opts…)` returns a `platform.Cursor` like the standard shapes, for `win.SetCursor`. The picture's pixels are points; `platform.HiDPI(img2x)` adds one at twice the size.
- It is a plain value, registered once, with no system handle: each window makes its system cursor the first time it shows one, and destroys them when it closes. Headless windows only record it, so tests read `win.Cursor()`.
- The desktop makes a GLFW cursor, from the HiDPI picture on a 2× screen on Windows and Linux; macOS, where GLFW 3.3 counts a cursor's pixels as points, shows the plain picture, scaled. The web sets the canvas's CSS cursor to a data URL, the HiDPI one beside it in an `image-set`.
- The UI's hover cursors still stand aside for any cursor the game sets.

## Alternatives

- A `*platform.Cursor` struct type for every cursor: the standard shapes would stop being constants, and every comparison and map of them in the UI and backends would change for one new case.
- An option per window instead of a value: each window would carry pictures, and switching between two picture cursors would need setters for each.

## Consequences

A game builds a picture cursor once and passes it anywhere a shape goes. Cursors made are never freed, which is fine at a handful a game; on macOS a HiDPI picture waits for GLFW 3.4 or a native cursor of our own.

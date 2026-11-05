# 0063. On the web, the page is the window's frame

- Status: Accepted
- Date: 2026-11-05
- Area: platform
- Owner: architect

## Context

In the browser a game's window is the page's canvas. It was sized once, at start, to the window's size, and SetFullscreen put the canvas itself full screen. The browser then stretched the canvas past the size the game read, so games drew in a corner and the pointer missed, and a resizable window stayed at its first size however large the page was.

## Decision

The page plays the part a desktop window's frame does (2c01b26c):

- Each frame, before the World's tick, the runner fits the canvas: to the page (innerWidth × innerHeight) while the page is full screen, whoever asked, or while the window is resizable (no `platform.FixedSize`, clamped to its MinSize); otherwise to the window's own size. On a change it sets the canvas's CSS size and its backing (× devicePixelRatio, or EMBER_SCALE), reconfigures the surface and syncs the window's size and framebuffer, as a desktop resize does. Fitting every frame, rather than on resize or fullscreenchange events, means no event can be missed.
- `SetFullscreen` asks for the whole page (document.documentElement) full screen, never the canvas, so a game's full screen and a page's own (a welcome page's Play) are the same thing.
- `platform.Window.SyncFullscreen(on)` records full screen changed by the player or the page (a browser's Escape), so `Fullscreen()` tells the truth.

## Alternatives

- Keep the canvas at the window's size and centre it on the page: a resizable game never grows, and full screen leaves bars the game can't draw in.
- Scale a fixed canvas to fit the page (CSS stretch or a letterboxed scale): the game would draw at one size and be shown at another, blurring pixel art and needing the pointer rescaled; desktop windows don't do that either.
- Full screen the canvas element: what we had, and the browser's own styling of a full-screen element is what broke the size.

## Consequences

A resizable game behaves in the browser as on the desktop, full screen included, and its Fit or camera code sees the real size. In the guide, every example without FixedSize now fills its Live page. Fixed windows keep their size until the page goes full screen. The rule is tested in Chrome (internal/chrome/fullscreen_test.go): a game's and a page's full screen, fixed and resizable windows at pixel ratios 1 and 2, a page that grows, and the pointer and keys after each.

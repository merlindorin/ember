# 0025. UI scales by subtree

- Status: Accepted
- Date: 2026-10-30
- Area: rendering
- Owner: architect

## Context

A UI laid out for a 720p window is small at 1440p or fullscreen, and games need some screens larger without redrawing their layout.

## Decision

- `ui.Scale(f)`, with `Scale()` and `SetScale`, on any element enlarges it and everything in it: sizes, padding, gaps, fonts, outlines and glows. Its parent gives it room f times its size.
- Layout inside runs at the unscaled size; drawing and the pointer go through the scale, so clicks land where things show.
- Text is rasterized at the scaled size, so it stays crisp, never a zoomed bitmap; a scaled element's tooltip is scaled too.
- `Rect()` is the element's rectangle on screen, through every scaled parent.
- There is no automatic mode yet: games set the scale per screen, such as the window's height over the height the screen was designed for.

## Alternatives

- **A window-wide scale:** one switch, but a HUD and a menu in the same window couldn't differ.
- **A theme option:** themes are shared and can't follow one window's size.

## Consequences

Screens scale independently and stay sharp. Code reading an element's laid-out rectangle inside a scaled subtree gets that subtree's frame; `Rect()` gives the screen.

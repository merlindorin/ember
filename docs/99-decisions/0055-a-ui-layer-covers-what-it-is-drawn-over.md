# 0055. A UI layer covers what it is drawn over, for the pointer

- Status: Accepted
- Date: 2026-11-03
- Area: core
- Owner: architect

## Context

A window can stack UI layers: ember/ui elements or twod widgets, then a
uiv2 Document drawn over them, such as the tileset tool's menus over its
tile grid and editor. Each layer arbitrated the pointer only among its
own parts, so a click on an open menu also reached the widget under it.
[0053](0053-a-modal-owns-its-windows-input.md) solved this for modals only.

## Decision

- `internal/wm` keeps one cover per window. A layer calls `Cover(w, win,
  e)` each frame one of its elements, not its page, is under the pointer.
  A uiv2 Document does so when its hit isn't html, body or
  `pointer-events: none`.
- Through the next frame only e and what is under it take win's pointer
  (`TakesPointer`). twod's `Hovered` refuses other entities and their
  `Mouse` is far away. Keys are untouched.
- A cover is stamped with its frame. A layer covering again this frame
  restamps it, so what is drawn after it, over it, is never covered by
  it.
- 0053's owner wins: a cover outside the owner counts for nothing.
  Releasing ownership also drops the owner's cover, so closing a modal
  still hands the pointer back the same frame.

## Alternatives

- Draw the tool's widgets through `uiv2.Canvas`. Its draw func runs only
  when invalidated, and gives the widgets no input of their own each
  frame.
- Have each layer hit-test the others. The layers don't know each other,
  and ember/ui, twod and every Document would each need the rest.

## Consequences

Layers stack for the pointer the way they show. The price is one frame
of lag: the widgets draw before the Document that covers them, so they
read the Document's claim from the frame before, and a fast move onto a
menu can reach the widget for that frame. That's acceptable for a tool. A
game that can't afford it has its widgets read the claim after the
Document draws.

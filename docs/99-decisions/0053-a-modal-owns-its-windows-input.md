# 0053. A modal owns its window's input; owners stack

- Status: Accepted
- Date: 2026-11-02
- Area: core
- Owner: architect

## Context

A window can show several UI layers at once: ember/ui elements, twod
widgets and several uiv2 Documents, such as the tileset tool's shell
and a file dialog drawn inline. Each layer arbitrated the pointer only
among its own parts. A click or key on a modal dialog also reached the
layers under it.

## Decision

- `internal/wm` keeps an owner stack per window. `Own(w, win, e)` puts
  `e` on top, and `Release` takes it out wherever it stands. The last to
  own wins, and the one below takes the input back at once.
- While a window has an owner, only the owner and the entities under it
  take its input. twod's `Canvas.Input()` hands any other entity being
  drawn an idle input: no keys, no buttons, the pointer far away.
  `Hovered` refuses that entity. ember/ui neither raises panels other
  than the one holding the owner nor moves focus there.
- An owned window's input also reads as captured (`PointerCaptured`,
  `KeyboardCaptured`) from the moment it is owned, so game systems stand
  aside too. Every frame re-captures it. Owners that left the World or
  are hidden (they or a parent) are dropped, and so are closed windows.
  An owner that only stops drawing keeps the window until it releases
  it.
- A uiv2 Document owns its window while a modal dialog of its is open.
  The owner entity is the one being drawn, `Canvas.Drawing()`: the
  Document, or the element hosting it. It owns on `ShowModal` and
  releases on close, without waiting for a frame. Shown again after
  being hidden with its modal open, it owns again.

## Alternatives

- Force kit dialogs into their own window (OwnWindow) in the tool. This
  dodges the problem for one tool, but inline dialogs and every other
  game screen still leak.
- Put the dialogs inside the inspector's Document, or have each Document
  capture on its own. Each layer only knows itself, so ember/ui, twod
  widgets and other Documents still take the clicks.

## Consequences

Any UI layer can be modal over the others in one window, at the cost of
a per-window lookup on each `Input()` and `Hovered` call. Captures hold
one frame after release, as every capture does.

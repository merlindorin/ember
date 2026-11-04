# 0027. Focus captures only what ui reads

- Status: Accepted
- Date: 2026-10-30
- Area: api
- Owner: architect

## Context

While a menu had active focus, ui captured the whole keyboard and every pad, so a screen that used other keys (I to equip, S to salvage) couldn't read them. The rule lived in the ui design, not in a decision.

## Decision

- Active focus captures only what navigation reads: the arrows, Tab, Enter, Space and Escape, and the pad's cross, left stick, South and East. Every other key and pad button reaches the game through `win.Input()`.
- A focused text input or area still captures the whole keyboard, as typing needs.
- `in.Captured(button)` says whether a key or pad button is captured, and the reads (`Down`, `Pressed`, `Value`, `Direction`…) leave captured ones up; `KeyboardCaptured()` means the whole keyboard only. The blanket pad capture goes.
- Capture holds from the frame it is set through the next, as before, and the raw reads ui navigates by ignore it.

## Alternatives

- **The game asks ui to release keys** it wants: every screen would list its keys twice.
- **Capture nothing,** the game filtering by focus itself: menus would move the hero again.
- **Two blanket flags,** keyboard and pads, as before: simple, but it is the problem.

## Consequences

Menus and games share the keyboard: navigation keys belong to the focused UI, the rest to the game. Code reading `GamepadCaptured()` moves to `Captured(button)`.

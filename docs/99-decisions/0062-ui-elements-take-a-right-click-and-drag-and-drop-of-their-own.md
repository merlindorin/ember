# 0062. ui elements take a right click and drag-and-drop of their own

- Status: Accepted
- Date: 2026-11-03
- Area: api
- Owner: architect

## Context

A game's inventory wants a right click on a slot (drop a stack) and a slot dragged onto another (swap them). ui's elements had neither: OnClick is the left button, and only windows, sliders and scrollbars dragged, inside ui.

## Decision

- **Right click.** `ui.OnContextMenu(fn)` runs when the right button is pressed and released on the same element, as the DOM's contextmenu, which opens no menu by itself. Like Back and keys ([0060](0060-back-and-key-callbacks-bubble.md)), it bubbles from the element to its ancestors, innermost first, so one handler on a grid serves its cells. The World event is `ui.ContextClicked{Node}`, emitted once; automation acts on it as "context".
- **Drag-and-drop, our own small model.**
  - A source is marked with the DOM's `draggable="true"` (`ui.Draggable()`).
  - A drag starts once the pointer moves 4pt with the left button held on a source, so a shorter movement stays a click. Escape, or a release over no drop target, cancels.
  - A target takes `ui.OnDrop(fn)`; its Event's `Dragged` is the source. The source may take `ui.OnDragEnd(fn)`, its Event's `Dropped` telling whether a target took it.
  - World events, emitted once each beside the callbacks, which stay the primary path: `ui.Dropped{Target, Dragged}` and `ui.DragEnded{Source, Dropped}`.
  - CSS reaches a drag through attributes, not new pseudo-classes: the source carries `data-dragging` while dragged, and the drop target under the pointer `data-drag-over`. The cursor shows the hand (no grab shape on the desktop yet; picture cursors, [0061](0061-picture-cursors-are-cursor-values.md), could give one). No ghost follows the pointer in this first version; a game draws one if it wants it.
  - Automation drags from one path to another ("drag").

## Alternatives

- HTML5 drag-and-drop as is (dragstart, dragover, drop, dataTransfer): its native ghost and system integration don't fit a game, and its event sequence is heavy for a slot swap.
- No drag in ui, each game building it over pointer input: every game solves the threshold, the cancel, the hit test and the drop-target state again, outside automation.

## Consequences

Inventories and editors drag and right-click without game-side pointer code, and tests drive both through automation. A ghost, keyboard or pad drag, and drag across Documents or windows are not covered; each would extend this decision. contextmenu has a native counterpart and is checked against Chrome; our drag is ui's own and is checked by interaction tests.

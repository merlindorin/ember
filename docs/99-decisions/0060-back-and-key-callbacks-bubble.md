# 0060. Back and key callbacks bubble from the focused element

- Status: Accepted
- Date: 2026-11-03
- Area: api
- Owner: architect

## Context

Back and keys land on the focused element, and [0057](0057-uiv2-elements-take-callbacks-over-the-world-events.md)'s callbacks didn't bubble, so a screen put `OnBack` on each of its elements.

## Decision

- `EventBack` and `EventKey` run the focused element's handlers, then each ancestor's up to the html element, innermost first. `e.Node` stays the focused element.
- `e.StopPropagation()` stops them; the Event carries an unexported pointer to the dispatch state.
- With nothing focused, a Back runs the body's and html's handlers, still with no World event, and the key is taken only when such a handler exists.
- Clicks, changes, inputs and the rest stay on their element.
- World events are emitted once, never per ancestor. An ancestor that left the Document during the dispatch is skipped.

## Alternatives

- Every event bubbles, as in the DOM: an `OnClick` on a container would fire for every child's click.
- A handler returns true to stop: `ui.Handler` has no form returning a bool.

## Consequences

A screen reacts to Back and keys once, on its root. A key handler on an ancestor must check `e.Node` when the target matters, and stop the event when it handled it.

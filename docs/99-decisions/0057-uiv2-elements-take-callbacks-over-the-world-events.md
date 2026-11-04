# 0057. uiv2 elements take callbacks over the World events

- Status: Superseded by [0060](0060-back-and-key-callbacks-bubble.md) for Back and keys, which bubble; its callbacks stand
- Date: 2026-11-03
- Area: api
- Owner: user

## Context

The user wanted callbacks on elements, as the DOM has them:
`uiv2.Button(uiv2.ID("save"), uiv2.Text("Save"), uiv2.OnClick(save))`.
Until now a game read every interaction from World events in a system.

## Decision

- Args OnClick, OnInput, OnChange, OnSubmit, OnToggle and OnClose, plus
  OnSort, OnKey, OnBack, OnPointerEnter and OnPointerLeave. Each is
  generic over `Handler`, which is `func()`, `func(Event)` or
  `func(*world.World, Event)`, so `OnClick(save)` takes a plain function.
- Callbacks sit on top of the events: every event is still emitted as
  before, and the element's callbacks for it run right after, in the
  Document's Update, in the order they were added, if the element is
  still in the Document. Events don't bubble.
- `Event` carries everything the typed events do: Clicks (2 or 3 for a
  double or triple click, newly counted on Clicked), a dialog's Value, a
  sort's Column and Descending, and a key's Key, Text and Command.
- Hover becomes events too, PointerEntered and PointerLeft (as
  mouseenter and mouseleave), for OnPointerEnter and OnPointerLeave.
- `uiv2.On(n, uiv2.EventClick, fn)` adds a callback live and returns its
  remover.

## Alternatives

- Separate names per form (OnClick, OnClickE, OnClickW): three times the
  API for what Go's inference does with one generic.
- Callbacks in place of events: systems, tests and automation read events,
  and one source of truth keeps both working.
- Bubbling, as the DOM's click does: no screen needs it yet, and it can
  be added without breaking these.

## Consequences

Screens can wire their actions where they build their elements. A
callback runs in the Document's Update, so what it changes shows the next
frame, like any other change a system makes there.

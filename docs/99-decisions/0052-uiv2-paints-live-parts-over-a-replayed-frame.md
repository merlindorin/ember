# 0052. uiv2 paints live parts over a replayed frame

- Status: Accepted
- Date: 2026-11-02
- Area: rendering
- Owner: architect

## Context

Under [0051](0051-uiv2-replays-its-last-frame-when-nothing-changed.md), a
frame that drew anything live repaints whole. A focused text field's caret,
an indeterminate progress or an open select's list then costs 35–89 µs a
frame for a few pixels.

## Decision

- Paint draws the tree without its live parts and records that. It keeps
  each live part's entry: its box, its place, its clip chain and transform.
  The live parts are a focused editable field's caret, an indeterminate
  progress's sliding segment and an open select's list.
- A frame where only time moved replays the recording, then paints those
  live parts alone on top, each under its own clips and transform. "Only
  time moved" means no input, no style, layout, scroll or size change and
  no transition: only the clock advanced (a caret's blink, an
  indeterminate slide). Typing, or moving the caret by keys, repaints
  whole, as before.
- A live part that something painted after it overlaps (by box or ink)
  can't go on top without covering that. Its frames repaint whole, as
  under 0051.
- Anything else that changes repaints whole and records again, by 0051's
  rule. F3 counts "ui replays + live" apart from "ui replays".

## Alternatives

- Repaint whole while anything is live (0051 as it stands): simple, but
  every focused form pays a full paint.
- Record per subtree and repaint the live box's range in place: exact
  stacking, but it is 0051's parked subtree ranges, with their range
  bookkeeping and batch splicing.

## Consequences

A focused field costs a replay plus its caret. The costs:
- an overlap check in the recorded pass, against what is painted after
  each live part, which falls back to whole repaints when one is covered;
- the live entries kept from frame to frame, with no allocation on a
  replayed frame. A test pins it: AllocsPerRun on a focused field's frame
  is 0 from uiv2.

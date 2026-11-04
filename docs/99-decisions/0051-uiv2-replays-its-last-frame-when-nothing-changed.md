# 0051. uiv2 replays its last frame when nothing changed

- Status: Accepted
- Date: 2026-11-02
- Area: rendering
- Owner: architect

## Context

uiv2 repainted every Document every frame: a static 1,000-element page cost
0.35 ms and the inspector 0.11 ms, against a 0.05 ms target. Most of it went
into laying out text that had not changed.

## Decision

- A Document that changed nothing since the last frame replays that frame's
  draw range: `draw.List` Mark/Record/Replay, wrapped by `twod.Canvas`, at the
  same origin and only under the same List state (transform, modulate,
  blend, snap, clip top). Otherwise it repaints and records again.
- Record refuses what can't be replayed: lit or classed vertices, masks,
  pass pictures and render textures. Glyph UVs are atlas texels, so atlas
  growth never stales a recording. F3 shows replays and repaints with
  their reasons.
- Changed frames re-shape only changed text runs: a shaped run (draw.Shape,
  `twod.Canvas.DrawShaped`) is kept per text fragment, keyed on text, face,
  size, tracking and device scale.
- Repainting only dirty subtrees is parked until numbers ask for it.

## Alternatives

- A per-run glyph cache inside draw, with no replay: cheaper changed frames,
  but every static frame still walks, decorates and emits the whole page.
- A fully retained scene graph (persistent GPU buffers per element): the
  fastest static and changed frames, but a second renderer beside twod's
  immediate lists, with its own batching, clipping and lifetime rules.

## Consequences

Static UI frames cost a copy of their vertices, and changed ones re-shape
only what changed. In exchange, uiv2 tracks whether a frame changed, and
draw keeps recordings, whose fonts and textures stay alive while recorded.
"Nothing changed" is a rule, not a hope. A frame repaints when:
- css saw a change (the root was dirty before it ran, even for a colour
  alone);
- layout moved boxes or a box scrolled;
- a transition ran in it or in the frame before;
- the last frame drew something live (a focused field's caret, an
  indeterminate progress, an open select's list);
- any input arrived;
- the canvas changed size or scale;
- Replay refused the List's state.

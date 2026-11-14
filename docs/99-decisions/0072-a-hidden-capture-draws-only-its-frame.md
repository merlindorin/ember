# 0072. A hidden capture draws only its frame

- Status: Accepted
- Date: 2026-11-14
- Area: rendering
- Owner: engine

## Context

A hidden still capture drew every frame up to the one it saves, and nobody sees those frames. On a software adapter
that is the whole cost: a game's 300-frame golden ran over 23 minutes on Windows' WARP, and its job timed out.

## Decision

- On the desktop runner, a hidden run that captures a still (`EMBER_HIDDEN` with `EMBER_CAPTURE=shot.png`) ticks
  every frame and draws only the captured one. A capture of frame N means the World after N ticks, drawn once.
- A frame the game reads back is still drawn: one where a window has light probes, answered from the frame drawn
  (`draw.ReadsBack`). Whatever else comes to read a drawn frame back joins that function.
- No other state kept from one draw to the next changed a picture. Static light bakes, glow textures, pass targets
  and atlases are made by the frame that needs them; scenes with particles, fades, effects and ui layout came out
  the same in the checks below. `golden`'s GPU test holds it on every adapter: four scenes, lit, probed, 3D and two
  windows, captured both ways must be the same bytes.
- `EMBER_CAPTURE_DRAW=all` draws every frame, for a scene that turns out to need it and for that test.
- GIFs, windows that show, hidden runs that capture nothing (benches, automation) and the browser draw every frame
  as before.

## Alternatives

- Raise the job's timeout: the cost grows with every scene, and an hour of CI checks nothing more.
- A fixed warm-up of drawn frames before the captured one: no scene needed one, and probes need all of them.
- Leave it to `golden` to ask for it: the guide's pictures and the ui pixel pages are the same captures.

## Consequences

A capture's time no longer grows with its frame number, but with its probes: a scene with probes in view draws as
before. `Window.Capture` asked on a frame not drawn is saved from the next one drawn. Checked the same bytes on Metal:
ember's 12 golden scenes, the 50 windowed examples at frame 40, the 77 ui pages, and Lantervale's 8 golden scenes.

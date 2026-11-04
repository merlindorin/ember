# 0038. A flow field may build in the background, one call late

- Status: Accepted
- Date: 2026-10-31
- Area: core
- Owner: architect

## Context

The horde's flow field ([0024](0024-a-horde-follows-one-flow-field.md)) is rebuilt over 128×128 cells whenever the hero enters a new cell. That cost 1.5–2 ms on those frames, and up to 4 ms under load.

## Decision

- `nav.Background()` is a FlowOption. SetTarget reads the grid on the caller's goroutine and builds the field on another into a second buffer. The next SetTarget waits for it and shows it.
- `Toward` leads by the field made for the call before, so the new field shows exactly one call later, never "when ready". Runs and tests repeat.
- The first field is built at once. Without the option nothing changes.
- The horde's chase uses it.

## Alternatives

- An incremental Dijkstra: when the goal moves by a cell, distances change almost everywhere, so a repair costs about as much as a rebuild.
- Time-slicing the build over several frames: still on the frame's thread, and the field lags several frames instead of one.
- A smaller area round the hero: cheaper, but it changes how far monsters are routed round walls.
- Precomputing each cell's open moves and skipping ways for cells in sight: the same field, but no gain. The Dijkstra dominates.

## Consequences

A cell-change frame now spends about 0.15 ms on the field. Monsters routing round walls follow a field one frame (16 ms) old, while those in sight of the hero still head straight at him. The default stays synchronous and unchanged.

# 0012. Nodes reuse the engine

- Status: Accepted
- Date: 2026-10-27
- Area: scenes
- Owner: engine

## Context

UI and 2D nodes could each get their own renderer, input and layout code.

## Decision

- UI nodes (`control`) lay out by anchors and containers, and draw through the immediate `ui`, so input, automation IDs and the GPU presenter are shared.
- 2D nodes draw through the same `ui.Canvas` batcher on the CPU, in one stream with the UI, ordered by canvas layer.
- Nodes are World entities: systems and nodes see the same data.

## Alternatives

- A retained UI and a second GPU pass for 2D: faster for huge scenes, but two more renderers to maintain.

## Consequences

About 10 000 sprites fit in a 60 fps frame. A GPU path for 2D waits for a game that needs more.

# 0002. Fixed-step simulation with prediction

- Status: Accepted
- Date: 2026-10-26
- Area: net
- Owner: engine

## Context

Results must not depend on frame rate, and play must feel instant at 150 ms of latency.

## Decision

- Gameplay runs in `FixedUpdate` at a fixed step. Rendering interpolates between steps.
- The client predicts what it owns, replays unacknowledged intents and blends corrections.
- Remote entities are shown slightly in the past, interpolated between snapshots and never extrapolated.

## Alternatives

- A variable step: simpler, but not reproducible and unstable at low frame rates.
- Extrapolating remote entities: less delay, but they overshoot every stop.

## Consequences

The simulation is deterministic and testable. Remote players appear about 100 ms behind the server.

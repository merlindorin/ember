# 0001. The server is authoritative

- Status: Accepted
- Date: 2026-10-26
- Area: net
- Owner: architect

## Context

ember is for online open worlds with competitive play, so a client must never decide what happens.

## Decision

- The server owns the simulation. Clients send intents and draw what the server replicates.
- Single-player runs the same server in process over `net.Pipe()`.

## Alternatives

- Client authority, or a peer-to-peer lockstep: cheaper servers, but cheating pays and late joins are hard.
- Separate solo and online code paths: simpler solo play, but two games to keep in sync.

## Consequences

There's one code path for every mode. Clients need prediction to feel responsive (0002).

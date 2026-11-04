# 0005. Transports are pluggable, QUIC is opt-in

- Status: Accepted
- Date: 2026-10-26
- Area: net
- Owner: architect

## Context

Games need an in-memory link for solo play, a stream for any network, and unreliable datagrams so lost packets don't stall movement.

## Decision

- `net` runs over any transport: a pipe, TCP or TLS.
- QUIC lives in `net/quic`, so only binaries that import it get quic-go. On links with datagrams, pose updates travel unreliably and everything else stays on the stream.

## Alternatives

- QUIC only: datagrams everywhere, but some networks drop UDP and every binary pays for quic-go.
- TCP only: simpler, but one lost packet holds back every later snapshot.

## Consequences

A game picks its transport and can fall back from QUIC to TLS.

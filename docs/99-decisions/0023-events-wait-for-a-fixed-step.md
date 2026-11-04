# 0023. Events wait for a fixed step

- Status: Accepted
- Date: 2026-10-30
- Area: core
- Owner: architect

## Context

Events lived two frames. Above twice the fixed rate, two frames can pass with no fixed step, so a FixedUpdate system missed what Update emitted: the demo's restarts did.

## Decision

- An event is readable by every phase's readers at least once. While the World has FixedUpdate systems, an event is dropped only once it is two frames old and a fixed step has begun after it.
- Each queue keeps one mark, where it stood when the last fixed step began; a World without FixedUpdate systems ages events by frames alone, as before.
- Readers keep their cursors, so nothing is read twice.

## Alternatives

- **Aging per phase,** each reader's phase with its own lifetime: exact, but the queue would need to know its readers' phases, and readers are plain values today.
- **A longer fixed lifetime,** say ten frames: simple, but still wrong above ten times the fixed rate, and it keeps events longer at every rate.
- **Telling games to emit only in FixedUpdate:** no engine change, but UI and input events come from Update.

## Consequences

At very high frame rates an event stays in memory until the next fixed step, at most one fixed period. Engine tests that age events by hand with EndFrame and no fixed systems behave as before.

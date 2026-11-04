# 0003. Tests need no GPU and no real time

- Status: Accepted
- Date: 2026-10-27
- Area: core
- Owner: engine

## Context

Engine tests ran slowly or flaked under parallel `-race` load when they used windows, goroutines or real clocks.

## Decision

- Every test runs without a GPU: the World runs frames headless, and renderers draw only while a window is presented.
- Network and prediction tests run on `net.Lag`, an in-memory link on a test clock. Only transport tests use real time.
- An example's test runs the program its `main` runs.

## Alternatives

- GPU and real-time integration tests: closer to production, but slow, flaky and impossible in CI.

## Consequences

`go test ./...` is fast and deterministic everywhere. GPU bugs are caught by running the examples.

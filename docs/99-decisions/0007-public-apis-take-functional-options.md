# 0007. Public APIs take functional options

- Status: Accepted
- Date: 2026-10-27
- Area: api
- Owner: user

## Context

The first example exposed `platform.Config{…}`. Config structs freeze field names into the API and hide defaults.

## Decision

- Public constructors take `WithX` options over an unexported config, and every option has a sensible default.
- No exported config struct is part of the API.

## Alternatives

- Config structs: fewer functions, but adding a field changes the API and zero values double as defaults.

## Consequences

Adding a setting never breaks callers. The remaining exported settings structs migrate to options.

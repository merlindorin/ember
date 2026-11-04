# 0004. Servers stay pure Go

- Status: Accepted
- Date: 2026-10-26
- Area: core
- Owner: architect

## Context

Game servers must be static binaries with no graphics libraries, and ember must not inherit any game's dependencies.

## Decision

- The pure packages (`world`, `scene`, `render`, `net`, `anim`, `gltf`, `fbx`, `asset`, `ui`, `node`…) never import cgo, `gpu`, `gfx` or a platform.
- `ember/go.mod` never depends on a game module.

## Alternatives

- One cgo build for client and server: simpler, but servers need GPU libraries and can't be distroless.

## Consequences

A server builds with `CGO_ENABLED=0`. Drawing code is split into a pure description (`render`) and a cgo backend (`gpu`).

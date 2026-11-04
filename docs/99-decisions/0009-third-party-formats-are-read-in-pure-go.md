# 0009. Third-party formats are read in pure Go

- Status: Accepted
- Date: 2026-10-27
- Area: assets
- Owner: engine

## Context

Asset Store packs ship FBX inside `.unitypackage` files, and the only way in was converting each file with Blender, which can't run on a server or in CI.

## Decision

- `fbx` reads binary FBX 7.x and `unitypkg` reads Unity packages and prefabs, with the standard library only. Both return `*gltf.Model`, so everything downstream is unchanged.
- Hostile input returns errors, never panics: lengths are bounded and the parsers are fuzzed.
- `asset.Library` opens every format the same way and reports missing files instead of failing.

## Alternatives

- Convert offline with Blender: no parser to own, but a manual step, and no Unity prefabs.
- A cgo FBX SDK: complete, but proprietary and breaks the pure-Go rule (0004).

## Consequences

Packs open directly in Studio and in games. We own parsers for formats without a spec.

# 0016. The editor and Studio are removed for now

- Status: Accepted
- Date: 2026-10-28
- Area: tools
- Owner: user

## Context

The API review keeps reshaping the engine (UI as entities, no engine resources), and the in-game editor and Studio had to be migrated at every step.

## Decision

- `ember/editor` and `ember/tools/studio` are removed, with Emberfall's editor panel and cheats.
- `gui` (ImGui) and `automation` stay, so a future editor has its foundations.
- The tools come back as a rework once the public API has settled.

## Alternatives

- Keep migrating both through every API change: no gap in tooling, but most of the work goes into tools the user plans to rebuild anyway.

## Consequences

There is no model viewer or scene editor until the rework. Format support (`fbx`, `unitypkg`, `asset`) stays in the engine and is tested without a viewer. Supersedes 0010.

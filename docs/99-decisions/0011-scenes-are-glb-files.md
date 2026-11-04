# 0011. Scenes are glb files

- Status: Accepted
- Date: 2026-10-27
- Area: scenes
- Owner: user

## Context

ember adds scenes: trees of nodes saved, instanced and edited as a whole. A custom format would duplicate what glTF already holds: nodes, meshes, materials, skins, animations, cameras and lights.

## Decision

- A scene is a self-contained `.glb`. Node types, properties, groups, signals and instances travel in the `EMBER_node` and `EMBER_scene` extensions.
- Scenes reference each other by stable ID, never by file path.
- Node types and properties have ember's own names, following the Go packages (`node3d.Camera`, `control.Button`), with no external engine as a reference.

## Alternatives

- YAML or JSON scenes: readable diffs, but a second format next to the models, and no other tool can open them.
- Another engine's scene format and names: familiar to its users, but it ties ember's API to a project it doesn't control.

## Consequences

Blender and any glTF viewer open ember scenes and ignore the extensions, which ember preserves.

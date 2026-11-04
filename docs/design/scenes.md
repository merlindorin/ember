# Scenes and nodes

Status: approved; phases 1 to 3 landed. It follows decisions 0011 (scenes are .glb files) and 0013 (every RPC uses Connect).

## Model

The World stays the runtime. A **node** is an entity (`*world.BasicEntity`, whose parent and children are the tree) carrying a `node.Node` component: `Name`, a stable `SceneID`, `Groups`, `ProcessMode`, `Owner`. A **node type** is a Go composition of components, registered under its package and type name (`node3d.Camera`). It has no class inheritance, but carries the properties of the components it embeds (`node3d.Node` adds `scene.Transform`…). Behaviour is Go only: a node names a registered Go component that implements any lifecycle interface. The new package `ember/node` (tree, lifecycle, signals, scenes, registry) is pure Go. Node types live in pure subpackages (`node/node3d`, later `node/control` and `node/node2d`), because `node` imports `gltf`, which imports `render`. A shard loads any scene.

## Core (unchanged from v1)

- **Hierarchy.** `AddNode`, `RemoveNode`, `Reparent`, `ParentNode`, `ChildNodes` and `QueueFree`, suffixed so they never clash with `world.BasicEntity`'s methods, which they keep in step. A node in the tree is in the world; `Reparent` within the tree keeps it there.
- **Names and paths.** Names are unique among siblings: a clash numbers the newcomer, `Enemy2`. `Get("Player/Hand")`, `Get("../Sun")` and `Get("/root/Level")` resolve relative and absolute paths. `node.Find[T]` is the typed form.
- **Groups.** Add, test and iterate (`tree.Group("enemies")`); `CallGroup` calls a Go method by name.
- **Lifecycle**, in tree order:
  - `EnterTree` runs parent first.
  - `Ready` runs children first, once (again after `RequestReady`).
  - `Process` runs in Update and `PhysicsProcess` in FixedUpdate, in tree order.
  - `ExitTree` runs children first.
- **process_mode**: inherit, pausable, when_paused, always, disabled, with `tree.Paused`.
- **Signals.** Signals are typed fields, `Pressed node.Signal[struct{}]`, named after the field in snake_case (`pressed`), or by a `signal:"name"` tag. They connect to Go methods by name, with the parameter types checked at load, or to closures. Flags: deferred, one_shot.
- **owner.** A scene saves the nodes its root owns; nodes added at runtime have no owner.

## Scene file: a self-contained .glb

A scene is a .glb with its meshes, textures, skins and clips embedded. There are no file paths anywhere. glTF carries what it already models:
- glTF nodes and their TRS;
- `mesh` plus `skin`, giving `node3d.Mesh` and `node3d.Skeleton`;
- `camera`, giving `node3d.Camera`;
- `KHR_lights_punctual`, giving `node3d.Sun` and `node3d.PointLight`;
- `animations`, giving the `node3d.Animator`'s library;
- materials, including emissive strength and unlit, which `gltf.Encode` already writes.

Two extensions carry the rest:
- **`EMBER_node`**, on each glTF node:
  ```json
  {"id": "1a2b3c4d5e6f7081", "type": "node.Timer", "groups": ["spawners"], "process_mode": "pausable",
   "behaviour": "game.WolfSpawner",
   "properties": {"wait_time": 30, "autostart": true}}
  ```
  `type` is the node class, `package.Type`; nodes without it import as `node3d.Node` or `node3d.Mesh` from their glTF content. `properties` hold the fields tagged `node:"prop"` that glTF doesn't model, named after the field in snake_case, with enums stored as words (`"fill|expand"`), written by reflection as `editor.Inspect` does. Unknown properties fail loudly with the node's path.
- **`EMBER_scene`**, at the file root:
  - `id` is the scene's stable ID (16 hex characters).
  - `connections` are `[{signal, from, to, method, flags}]`, with `from` and `to` as node IDs.
  - `instances` are `[{node, scene, overrides, editable_children}]`. `node` is the glTF node standing for the instance, and `scene` is a scene ID.
  - `inherits` names the base scene's ID.

**References are IDs, never filenames.** An instance names its scene by ID. The scene is either **embedded**, as a subtree of the same .glb marked `EMBER_node.instance_of`, or **referenced**, resolved through `asset.Library` from an index of scene IDs to the files that hold them, which the library builds when it scans. Overrides address nodes inside the instance by node ID, with a path kept beside it for readability. Inherited scenes work the same way: `inherits` names the base by ID, and the file stores only the differences.

**Interop.** Unknown extensions and extras are kept and written back on save. Ember extensions are listed in `extensionsUsed` but never in `extensionsRequired`, so Blender and other glTF tools open the file and simply ignore Ember's data. `gltf.Load` and `gltf.Encode` are extended, not replaced: the model keeps a raw-extension map per node and per document, and `node.Load` and `node.Save` build on top of them.

## Node types

- **Phase 1 (3D):**
  - `node.Empty`, `node3d.Node`, `Mesh`, `Camera`, `Sun`, `PointLight`, `Environment` (render.Environment and render.Post);
  - `Skeleton`, `Socket` (anim.Socket), `Animator` (anim.Animator; `animation_finished`);
  - `Marker`, `node.Timer` (`timeout`);
  - `Replicator` (net.Network on its root_path; it syncs the pose and Stateful). Nodes placed by a level take their `net.ID`s from its scene IDs, and clients adopt them instead of spawning them.
- **Phase 2 (UI, Control):**
  - Layout: anchors and offsets, size flags, custom_minimum_size, and a Theme resource stored in `EMBER_scene.resources`. Theme items are found by the control's own type (`Button`), through its theme, its ancestor controls' themes, the project theme and the default.
  - Containers: Row and Column, Grid, Margin, Center, Frame and Scroll.
  - Controls: Label and RichText (text keys from the game's table), Button, Check, TextInput, Slider, Panel, ColorRect, Image, NinePatch, Progress, ImageProgress.
  - Signals: pressed, toggled, text_changed, text_submitted, value_changed.
  - These retained nodes draw through the immediate `ui` each frame, with the node path as the automation ID.
  - UI nodes are glTF nodes without meshes; their rect lives in `EMBER_node.properties`.
- **Phase 3 (2D):** `node2d.Node`, `Sprite`, `AnimatedSprite` (with `Frames`), `Camera`, `Line`, `Polygon`, `Marker`, `Tint`, and `canvas.Layer`. Tile maps come later. `gpu` needs:
  - a 2D pass with an orthographic camera per layer;
  - a sprite batcher (atlas, z_index, y_sort);
  - polygon and polyline fills;
  - a modulate uniform.

## Imports, Studio and live inspection

- **Imports.** A glTF or FBX import is already a scene: it gains `EMBER_node` IDs and types on save.
- **Studio** opens and saves scenes as .glb files. The hierarchy panel becomes the node tree, where you can rename, reparent, add a typed node, instance a scene and toggle editable children. The inspector edits registered properties.
- **SceneService** (Connect, `ember.scene.v1`) is served by any ember app, game or shard, when automation is on, on the same h2c server and token. It offers:
  - `GetTree`: nodes with ID, name, path, type, behaviour, groups, process mode and instanced scene, to a depth;
  - `WatchTree`: a stream of added, removed and changed nodes under a path, the first message listing them all;
  - `GetNode`: properties by name, and signals;
  - `SetProperty`;
  - `Isolate`: pause everything but a subtree, for inspection; an empty path ends it;
  - `Snapshot`: a subtree, or the nodes within a radius of a point, as .glb bytes streamed in chunks (`node3d.Snapshot`, through `node.Tree.Resolve`).

  Studio reads a running game and a file the same way: a snapshot is just a scene.

## Deferred

- Adding nodes deeper inside an instance (editable children), and renaming nodes inside an instance.
- Phase 2 leftovers: mouse_filter, focus and Tab order, per-control theme overrides, NinePatch draw_center, BBCode [url].
- Phase 3 leftovers: tile maps, Polygon textures and vertex colors, camera drag margins and rotation smoothing, a GPU-transform path for large canvases.

## Decided

1. glTF node TRS maps one-to-one to `scene.Transform`; UI and 2D nodes use identity TRS plus their own properties.
2. A missing or duplicate scene ID fails loudly, naming the ID and every file involved.
3. One .glb per prefab, referenced by ID, is the norm; only small one-offs are embedded.
4. The tree runs in its own systems (Update and FixedUpdate at priority 0); engine systems keep theirs.
5. `Snapshot` covers a subtree or a radius, streamed in chunks.
6. Names are Ember's own: types are named for what they are, properties after their Go fields, and enums by words.

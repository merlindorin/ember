# Guide plan

Status: the guide's own rules and plan, kept by the guide's writer; not an engine proposal. The guide grows one reviewed example at a time ([decision 0014](../99-decisions/0014-the-guide-teaches-by-example.md)): a page appears when its example is approved, a section when its first page does. This table is the map until then.

## Writing a page

- `# Title`, then one or two sentences: what the example shows.
- Then one `##` step per idea, named for what it does ("Sending one", "A camera per window"): a sentence, then the file it is about, as a link alone on its line, `[cooling.go](../../examples/04-events/cooling.go)`. Unchanged files go under "As before". Every file of the example must appear; the test checks it.
- End with `## Try it` and the example's folder as a link alone on its line, `[04-events](../../examples/04-events)`. The site shows the `go run` line and the output there.
- A screenshot goes right after the first sentences, as a plain image next to the page. For a window, take the raw capture, `EMBER_HIDDEN=1 EMBER_CAPTURE=docs/NN-section/name.png EMBER_CAPTURE_FRAME=30 go run ./examples/NN-name` (extra windows land in `name-2.png`…), halve it to the window's size in points, and give the image a title: `![The scene](scene.png "ember · scene")`. The site draws the window frame around it with that title, every capture at the same scale; several on one line overlap like windows on a desktop, the last one in front.
- The page's first titled capture is the example's whole window, at its size (the same shape as its `platform.Size`): the site then gives it Live and Screenshot tabs, running the example in the page where WebGPU is (7a6a3bd2). A page whose capture is another shape, such as a cropped kit card, gets the "Run in your browser" button under Try it instead.
- An animation is a GIF taken the same way: `EMBER_CAPTURE=name.gif` records `EMBER_CAPTURE_FRAMES` frames (default 90), one every `EMBER_CAPTURE_EVERY` steps (default 2, so 30 fps), from `EMBER_CAPTURE_FRAME` (default 1), already at the window's size in points. Every capture script separates its entries by ";". `EMBER_CAPTURE_KEYS="right:0-75;left:75-150"` holds keys in the primary window during those steps, so the example's code stays as it is. A capture or a hidden run reads no real device: no keyboard, mouse, trackpad or pad at the machine reaches the game and no pad rumbles, only these scripts play, so a capture gives the same frames whoever sits there. `EMBER_CAPTURE_MOUSE="320,192@0;140,120@40;left:30-40"` moves the cursor through those window points at those steps, in straight lines, holds mouse buttons (left, right, middle, back, forward) and turns the wheel (`wheel:-3@120`). `EMBER_CAPTURE_TEXT="Heron@150"` types that text from that step, a character every four steps. `EMBER_CAPTURE_PAD="south:30-40;leftstick:0,0@0;leftstick:1,0@30;righttrigger:1@60;1/connect:90-180"` plays pads: buttons by position held over steps, sticks and triggers moving between values, `connect` when a pad is plugged in, `1/` for pad 1. A stick or trigger rests until its first keyframe; one keyframe leans it from that step on (`leftstick:1,0@30` pushes right from step 30, a press a menu reads), so a value meant from the start takes `@0`. `EMBER_CAPTURE_SIZE="640,360@0;900,480@60"` resizes the primary window through those sizes, for a layout that responds; frames smaller than the largest sit on a dark desk. A GIF keeps only what changes from frame to frame, so still scenes cost little and a moving camera costs most. Colours are dithered with a fixed pattern, so gradients don't band; a smooth glow that moves costs more, so keep such recordings short (20 fps with `EMBER_CAPTURE_EVERY=3`). Keep it to 2–3 seconds and under about 1 MB; the page shows it like a capture, `![The hero walking](walk.gif "ember · sprite sheet")`, and the site serves it as it is.
- Pixel art in examples and on pages follows two rules. World art is drawn whole at one whole-number scale, 2 points per art pixel for 0x72's tiles, never resized, cropped or sliced; tiles stand bottom-left on their cell as Tiled draws them, and tall pieces rise above it (decision 0034). A UI icon is drawn at the largest whole multiple of its own pixels that fits its slot at the current `ui.Scale`, at least 1×, with nearest filtering, centred in the slot, never at a fractional size. The nine-patch page is the one place art is sliced, on purpose, for frames.

## Plan

Next: 2D only, as the user reviews the public API through the guide: 20-sprite-sheet, 21-lines-and-shapes, 23-2d-camera, then nine-patch, text, many sprites, bitmap font and clipping; tile maps, collision, y-sorting and Tiled maps. 3D and networking wait for their API reviews; the 3D API will be redone, so its examples (materials and lights, procedural meshes) were removed rather than kept for a page. The make-a-game tutorial and the showcases (90–93) were removed on 2026-10-30; the demo (`go run ./demo`) is the full game. The theme waits for the UI's step 5; translations are not planned.

| Example | Section | Page | Shows | Status |
|---|---|---|---|---|
| 01-hello-cube | getting-started | Hello cube | a window, a mesh, a camera | published |
| 02-entities-and-systems | world | Entities and systems | components, an entity, a system, `On` | published |
| 03-resources | world | Resources | `Insert`, `world.Resource` | published |
| 04-events | world | Events | `world.Emit` and `world.Reader`: one system tells another without knowing it | published |
| 05-children | world | Parents and children | `AppendChild`: children are added and removed with their parent | published |
| 06-views | world | Views | `AddView`: a group kept in sync but never updated | published |
| 07-fixed-step | world | Fixed step | `FixedUpdate` against `Update`, frames driven by `w.Frame` | published |
| 08-time | world | Time | `w.Time().Scale` for half speed and pause, `Alpha` to draw between fixed steps | published |
| asset-library | assets-and-animation | The asset library | `asset.Library` over a folder, a zip or a `.unitypackage`; glb, FBX, OBJ, STL and TGA; import reports | planned |
| grips | assets-and-animation | Grips | rig profiles, a holder with a second hand by IK, reach and clipping checks | planned |
| retarget | assets-and-animation | Retargeting | playing one rig's clips on another with `anim.Retarget` | planned |
| root-motion | assets-and-animation | Root motion | locomotion clips with `Animator.RootMotion`, moving the entity by `Motion()` | planned |
| dedicated-server | networking | A dedicated server | a headless, cgo-free server with a separate client | planned |
| net-pipe | networking | Server and client in one process | `net.NewServer` and `net.NewClient` added as plugins the game keeps; `Replicated`, `Stateful` and `OwnerStateful`, intents and client prediction | planned |
| net-quic | networking | QUIC | QUIC with a certificate pin, datagram snapshots and the TLS fallback | planned |
| lockstep-tests | networking | Testing games | `RunFrames` and `net.Lag`: deterministic, GPU-free tests | planned |
| crafting | open-world | Crafting | the craft registry, stacking inventories and recipes with typed errors | planned |
| spatial-queries | open-world | Spatial queries | the spatial grid and indexer for proximity queries | planned |
| terrain-streaming | open-world | Terrain streaming | `terrain`, biomes, the chunk streamer, scatter and noise | planned |
| materials-and-lights | rendering | Materials and lights | textured, transparent, emissive and double-sided materials, a sun with shadows, two point lights, fog and bloom from `render.NewEnvironment` | planned; the 3D API will be redone, so its example was removed |
| picking-gizmos | rendering | Picking and gizmos | `render.View` picking, world-to-screen, gizmos and billboards | planned |
| procedural-meshes | rendering | Procedural meshes | primitives chained with `Translate`, `Rotation`, `Scale` and `Tint`, joined with `Merge`, faceted with `Flat`, built once and shared | planned; the 3D API will be redone, so its example was removed |
| 09-window | windows | A window | `platform.NewWindow` with its options, a label as the window's child | published |
| 10-multi-window | windows | Multiple windows | windows as entities; an orbit camera, a fixed camera and labels as windows' children, each reading and drawing in its own window | published |
| 19-sprites | 2d | Sprites | `twod.LoadTexture` with `Pixelated`, sprites made with `At`, `Scale`, `Rotation`, `Flip` and `Tint` | published |
| 20-sprite-sheet | 2d | Sprite sheets | a sheet cut with `Strip`, clips from `NewAnimations` given with `Animate`; a hero walking with the arrow keys: `Move`, `Play`, `SetFlip`, `Stop` | published |
| 21-lines-and-shapes | 2d | Lines and shapes | `NewLine` with `Width`, `Joints` (`Miter`/`Bevel`/`RoundJoint`), `Caps` (`Flat`/`Square`/`RoundCap`) and `Closed`; a concave `NewPolygon` | published |
| 23-2d-camera | 2d | A 2D camera | a side-scroller: `NewCamera` with `Follow`, `Smoothing` and `Bounds`; layers in order, a `Parallax` layer, a `Fixed` HUD layer | published |
| 39-keyboard | input | Keyboard | keys as places (WASD on any layout), `Direction`, Shift either side to run, `Text()` for a typed name, Backspace, paste from the clipboard | published |
| 40-mouse | input | Mouse | `Mouse()` through `ScreenToWorld`, dragging crates, the wheel zooming, `SetCursor`, Back and Forward undoing moves | published |
| 41-actions | input | Actions | an action map (`input.NewActions`, `Bind`, `Direction`, `Pressed`) over keys, mouse and a pad (position-named buttons, sticks as analog buttons, `Gamepad(id)` per player, connect events) walking the hero and planting with a click or Enter, saved as JSON; the pointer through `ScreenToWorld`, the wheel zooming, a hidden cursor; text, clipboard, drops, UI capture and simulated input in prose | published |
| 42-gamepads | input | Gamepads | a pad visualiser drawn from the window's input (buttons by position with a PlayStation/Xbox/Nintendo table, analog triggers and sticks, the deadzone), the hero walked by stick or cross through an action map, `Gamepads()`/`Gamepad(id)` and connection events; GIF from `EMBER_CAPTURE_PAD` | published |
| 43-touch | input | Touch | trackpad fingers drawn on a pad: `Touches()` with ID, Source, At (0–1 on the pad, not window points) and Phase, `TouchStarted`/`TouchEnded`; macOS only | published |
| 44-gestures | input | Gestures | a crate pinched (`Pinch`), turned (`Rotation`), panned (`Pan`) and reset by `Swipe`; scroll still as `Wheel`; GIF without a swipe until it's verified on hardware | published |
| 45-play-a-sound | audio | Play a sound | `audio.Load` a WAV clip, `audio.NewMixer` as an entity, `Play` with `Pitch` and `Pan`, `Voices`, `VoiceFinished`; the silent output and `audio.NewRecorder` in the section page | published |
| 46-music-and-positional | audio | Music and positional sound | `audio.Open` Ogg streams, buses (`WithBus`, `OnBus`), `PlayMusic` with `FadeIn` and `Crossfade`, an `audio.Listener` on the hero and `audio.Emitter`s with `Distance`/`Rolloff`, `Heard()` on a HUD | published |
| 47-audio-effects | audio | Effects | a reverb on the sfx bus blended from corridor to hall, a low-pass on an emitter swept open by a tween; effects on buses or voices, chain order, setters that glide | published |
| 22-nine-patch | 2d | Nine-patch | `NewNinePatch` with `Margins` and `Size`, placed by a `Pivot(0, 0)` corner and scaled as pixel art | published |
| 36-text | text | Text | `NewText` with `Typeface` (the engine's faces and `NewFace`/`LoadFace` from a TTF), `FontSize`, `Color` and `Wrap`; `SetText` every frame | published |
| 37-bitmap-font | text | Bitmap font | `LoadBitmapFont` from a PNG with `Cell` and `Charset`, `NewBitmapText` with `Glyphs`, `Scale` and `Shadow`, the default font; `SetText` every frame | published |
| 38-emoji | text | Emoji | fallback faces (`With(Fallback(…))`), colour pictures from `LoadEmoji` with `Emoji(set)`, grapheme clusters measured, wrapped and deleted as one, an emoji-capable text input; what monochrome can't do and the licences | published |
| 25-clipping | 2d | Clipping | a layer clipped with `ClipTo(rect)`: the same scrolling ticker free and clipped to a framed strip | published |
| 24-many-sprites | 2d | Many sprites | a bunnymark: sprites added at runtime to a layer, moved every frame, a counter on a layer above, `Stats().FPS` | published |
| 26-tilemap | 2d | Tile maps | `NewTileset` with `TileAnimation`, `NewTileMap` with `ASCII(level, legend)` and `CellSize`; only the cells in view are drawn | published |
| 27-collision | 2d | Collision | `Solid` tiles, `collide.NewSpace`/`AddSolid`, bodies as children (`Box`, `Offset`, `Trigger`), `space.Move` stopping and sliding, `collide.BodyEntered` opening a door with `Set` | published |
| 28-y-sort | 2d | Y-sorting | tall tiles (`Tall`, `TallIn`) in a `SortByY` layer with the hero, solid trunks | published |
| 29-tiled | 2d | Tiled maps | `LoadTiled` of a .tmj and its .tsj: `Layers()` as tile maps and solids, `Objects(layer)` turned into the hero and pillars by `Type` | published |
| 30-pixel-perfect | 2d | Pixel-perfect | a camera's `Resolution(320, 180)` scaled up by a whole factor with bars, `Upscale`, `FitWindow`, fixed layers at window resolution, `ScreenToWorld` | published |
| 31-blend-modes | 2d | Blend modes and masks | `Blend(Multiply)` night layer, `MaskedBy`/`AsMask` lantern light, `Blend(Add)` fireflies, a glow texture made in code | published |
| 32-render-to-texture | 2d | Render to texture | `NewRenderTexture`, a camera `Into` it at `Zoom(0.5)` for a minimap shown as a sprite on a fixed layer; `SnapToPixels` | published |
| 33-2d-shader | 2d | 2D shaders | ripples where you click: `render.NewEffect` on the meadow's layer with `twod.WithEffect`, its `uRipples` set with `SetVec4s` from Go every frame; a fixed help layer stays undistorted | published |
| 34-particles | 2d | Particles | a campfire of `NewParticles` emitters (`Rate`, ranges, `Direction`, `ColorOverLife`/`SizeOverLife`, `Blend(Add)`) and click bursts (`Burst`, `OneShot`, `Drag`) removed on `ParticlesFinished` | published |
| 35-easing | 2d | Easing and paths | the `ease` families side by side, one lane each, through `ease.NewTween` (`PingPong()`, `Loop()`), and a walker on `geom.ClosedCatmullRom` | published |
| scene-3d | scenes | 3D scenes | a 3D level saved to `.glb` and reloaded: `node3d.Mesh`, `Camera`, `PointLight`, `Sun`, `Environment` and `Marker` | planned |
| scene-character | scenes | Animated characters | `node3d.Skeleton`, `Animator` and its signals, a `Socket` holding a sword | planned |
| 11-ui-elements | ui | UI elements | `ui` elements as entities built with options (`NewLabel`, `NewButton`, `NewRow`, `NewColumn`, `NewPanel`), `Add`, and the `Clicked` event changing a label | published |
| 12-layout | ui | Layout | a main menu filling a resizable window: a padded `Column` with `Expand` spacers, a `Grid`, a footer `Row` that `Fill`s, `Anchor` as the pivot | published |
| 13-controls | ui | Controls | a settings panel: `TextInput`, `Slider`, `Check`, `Progress`, `Scroll`; their events `Toggled`, `ValueChanged`, `TextChanged`, `Submitted` in a table; a value set from Go | published |
| 14-text-editing | ui | Text editing | a notes app: `TextInput` and `TextArea` with selection, word and line moves, clipboard, undo, scrolling; their options; a key table for macOS and Windows/Linux; `TextChanged` with `Area`; IME not yet | published |
| 15-themes | ui | Themes | one dialog in four looks: `NewTheme` with `For[T]` and box styles, `LoadTheme` from JSON (linking ui/testdata/theme.json), `SetTheme`, and a nine-patch skin | published |
| ui-over-3d | ui | A HUD over 3D | a HUD over a 3D scene: a `ui.Progress` bound to game state and a label pinned to an entity with `ui.Follow` | planned |
| scene-instances | scenes | Instances and inheritance | a prefab instanced twice with overrides, an inherited scene and `Library.Scene` | planned |
| scene-live | scenes | Live editing | a running game's scene tree inspected and changed through `SceneService` on the automation server | planned |
| scene-tree | scenes | The scene tree | a node tree built in code: names, paths, groups, `CallGroup`, the lifecycle order, process modes and `QueueFree` | planned |
| signals-timer | scenes | Signals and timers | typed signals, `ConnectMethod`, the deferred and one-shot flags, `Timer` and a Go behaviour | planned |
| automation | tools | Automation | `automation.New` passed with `ember.Automation`, game intents with `auto.Define`, recorded UI IDs and headless screenshots | planned |
| 17-custom-elements | ui | Custom elements | a game's own element: `ui.Base`, `ui.Build`, `Draw` on a `twod.Canvas` in the element's own rectangle, `Record` for automation, laid out with built-in elements | published |
| 18-testing-ui | ui | Testing UI | a settings form filled by `go test`: input simulated between `RunFrames`, then automation acts (`Recording`, `Queue`, `ActType`/`ActSet`/`ActPress`) by `Path()` IDs, and `Snapshot`; the test file shown | published |
| input-cameras | ui-and-input | 3D cameras and gamepads | gamepads, sticks with deadzones; `camera.NewOrbit`, and the Follow and Fly cameras (keys and mouse are in the Input section) | planned |

# Decisions

The choices that shape ember, one short file each: what was decided, what else was possible, and what follows.

| # | Decision | Area |
|---|---|---|
| 0001 | [The server is authoritative](0001-the-server-is-authoritative.md) | net |
| 0002 | [Fixed-step simulation with prediction](0002-fixed-step-simulation-with-prediction.md) | net |
| 0003 | [Tests need no GPU and no real time](0003-tests-need-no-gpu-and-no-real-time.md) | core |
| 0004 | [Servers stay pure Go](0004-servers-stay-pure-go.md) | core |
| 0005 | [Transports are pluggable, QUIC is opt-in](0005-transports-are-pluggable.md) | net |
| 0006 | [The World is the centre](0006-the-world-is-the-centre.md) | core |
| 0007 | [Public APIs take functional options](0007-public-apis-take-functional-options.md) | api |
| 0008 | [Nothing is styled by default](0008-nothing-is-styled-by-default.md) | rendering |
| 0009 | [Third-party formats are read in pure Go](0009-third-party-formats-are-read-in-pure-go.md) | assets |
| 0010 | [Studio is the official tool](0010-studio-is-the-official-tool.md) | tools |
| 0011 | [Scenes are glb files](0011-scenes-are-glb-files.md) | scenes |
| 0012 | [Nodes reuse the engine](0012-nodes-reuse-the-engine.md) | scenes |
| 0013 | [Every RPC uses Connect](0013-every-rpc-uses-connect.md) | tools |
| 0014 | [The guide teaches by example](0014-the-guide-teaches-by-example.md) | docs |
| 0015 | [The engine exposes no resources](0015-the-engine-exposes-no-resources.md) | api |
| 0016 | [The editor and Studio are removed for now](0016-the-editor-and-studio-are-removed-for-now.md) | tools |
| 0017 | [UI is built on 2D](0017-ui-is-built-on-2d.md) | rendering |
| 0018 | [The 2D API follows a few rules](0018-the-2d-api-follows-a-few-rules.md) | api |
| 0019 | [Options read back and pair with setters](0019-options-read-back-and-pair-with-setters.md) | api |
| 0020 | [Sound is pure Go behind one device](0020-sound-is-pure-go-behind-one-device.md) | core |
| 0021 | [A 2D view resizes by its camera](0021-a-2d-view-resizes-by-the-camera.md) | rendering |
| 0022 | [A camera names the layers it draws](0022-a-camera-names-the-layers-it-draws.md) | rendering |
| 0023 | [Events wait for a fixed step](0023-events-wait-for-a-fixed-step.md) | core |
| 0024 | [A horde follows one flow field](0024-a-horde-follows-one-flow-field.md) | core |
| 0025 | [UI scales by subtree](0025-ui-scales-by-subtree.md) | rendering |
| 0026 | [Pads rumble through the platform's haptics](0026-pads-rumble-through-platform-haptics.md) | core |
| 0027 | [Focus captures only what ui reads](0027-focus-captures-only-what-ui-reads.md) | api |
| 0028 | [A soft crowd spreads by density](0028-a-soft-crowd-spreads-by-density.md) | core |
| 0029 | [A flat tile draws its ground](0029-a-flat-tile-draws-its-ground.md) | rendering |
| 0030 | [Tiled flips are honoured](0030-tiled-flips-are-honoured.md) | assets |
| 0031 | [A batch draws several textures](0031-a-batch-draws-several-textures.md) | rendering |
| 0032 | [Lights see exact visibility polygons](0032-lights-see-exact-visibility-polygons.md) | core |
| 0033 | [A pictured layer can draw at a lower resolution](0033-a-pictured-layer-can-draw-at-a-lower-resolution.md) | rendering |
| 0034 | [Every tile draws whole](0034-every-tile-draws-whole.md) | rendering |
| 0035 | [A tile collides by its shapes](0035-a-tile-collides-by-its-shapes.md) | core |
| 0036 | [A solid tile blocks its cell for paths](0036-a-solid-tile-blocks-its-cell-for-paths.md) | core |
| 0037 | [Lights read the light maps](0037-lights-read-the-light-maps.md) | core |
| 0038 | [A flow field may build in the background, one call late](0038-a-flow-field-may-build-in-the-background.md) | core |
| 0039 | [ui styling: boxes, classes and a cascade](0039-ui-styling-boxes-classes-and-a-cascade.md) | api |
| 0040 | [Routing follows where a body fits, not whole solid tiles](0040-routing-follows-where-a-body-fits.md) | core |
| 0041 | [One renderer on WebGPU, through ember's own binding](0041-one-renderer-on-webgpu.md) | rendering |
| 0042 | [2D lights and shadows on the GPU](0042-2d-lights-and-shadows-on-the-gpu.md) | rendering |
| 0043 | [Standard tile properties, checked by the loader](0043-standard-tile-properties.md) | assets |
| 0044 | [Names made consistent](0044-names-made-consistent.md) | api |
| 0045 | [ShadedBy draws a light picture per set of groups](0045-shaded-by-draws-a-light-picture-per-set-of-groups.md) | rendering |
| 0046 | [Silhouettes are masks subtracted per light](0046-silhouettes-are-masks-subtracted-per-light.md) | rendering |
| 0047 | [Lights past a camera's budget fade out by priority, then distance](0047-lights-past-a-camera-budget-fade-out-by-priority-then-distance.md) | rendering |
| 0048 | [A light Affects layers through a picture per set of lights left out](0048-a-light-affects-layers-through-a-picture-per-set-of-lights-left-out.md) | rendering |
| 0049 | [UI is modern HTML elements and a CSS subset, written from Go](0049-ui-is-modern-html-elements-and-a-css-subset-written-from-go.md) | api |
| 0050 | [The renderer encodes with the exact sRGB curve and blends encoded](0050-the-renderer-encodes-with-the-exact-srgb-curve-and-blends-encoded.md) | rendering |
| 0051 | [uiv2 replays its last frame when nothing changed](0051-uiv2-replays-its-last-frame-when-nothing-changed.md) | rendering |
| 0052 | [uiv2 paints live parts over a replayed frame](0052-uiv2-paints-live-parts-over-a-replayed-frame.md) | rendering |
| 0053 | [A modal owns its window's input; owners stack](0053-a-modal-owns-its-windows-input.md) | core |
| 0054 | [ember runs in the browser through WebAssembly and WebGPU](0054-ember-runs-in-the-browser-through-webassembly-and-webgpu.md) | rendering |
| 0055 | [A UI layer covers what it is drawn over, for the pointer](0055-a-ui-layer-covers-what-it-is-drawn-over.md) | core |
| 0056 | [uiv2 images filter in encoded sRGB, as CSS does](0056-uiv2-images-filter-in-encoded-srgb.md) | rendering |
| 0057 | [uiv2 elements take callbacks over the World events](0057-uiv2-elements-take-callbacks-over-the-world-events.md) | api |
| 0058 | [Rounded clips through a per-batch rounded rect](0058-rounded-clips-through-a-per-batch-rounded-rect.md) | rendering |
| 0059 | [ember/ui is uiv2: the old package removed, uiv2 renamed ui](0059-ember-ui-is-uiv2.md) | api |
| 0060 | [Back and key callbacks bubble from the focused element](0060-back-and-key-callbacks-bubble.md) | api |
| 0061 | [Picture cursors are Cursor values](0061-picture-cursors-are-cursor-values.md) | api |
| 0062 | [ui elements take a right click and drag-and-drop of their own](0062-ui-elements-take-a-right-click-and-drag-and-drop-of-their-own.md) | api |
| 0063 | [On the web, the page is the window's frame](0063-on-the-web-the-page-is-the-windows-frame.md) | platform |
| 0064 | [A server World is ember.New without presenting](0064-a-server-world-is-ember-new-without-presenting.md) | net |
| 0065 | [2D entities replicate on the ground plane](0065-2d-entities-replicate-on-the-ground-plane.md) | net |
| 0066 | [A Tiled map loads only the tilesets it uses](0066-a-tiled-map-loads-only-the-tilesets-it-uses.md) (superseded by 0067) | assets |
| 0067 | [A Tiled map reads only the tilesets it uses](0067-a-tiled-map-reads-only-the-tilesets-it-uses.md) | assets |

## Writing one

- Only engine decisions belong here: a real choice between alternatives that would be costly to reverse. Implementation details belong in package docs, and design notes in [design](../design).
- Record a decision once it has settled, not at every step of a change.
- Copy [TEMPLATE.md](TEMPLATE.md) to `NNNN-short-title.md` with the next number, and add a row above, in the same commit as the change.
- Keep it under a screen. If there was no alternative, it isn't a decision.
- Never rewrite an accepted decision: write a new one and mark the old one `Superseded by [NNNN](…)`.

# 0017. UI is built on 2D

- Status: Accepted
- Date: 2026-10-28
- Area: rendering
- Owner: user

## Context

2D drawing (`twod`) and the UI each had their own drawing layer, transforms and draw order, so HUD, sprites and backdrops could not sort together, and custom UI drew with a different API from 2D games.

## Decision

- `twod` is all 2D drawing: shapes, images, nine-patches, text, layers, order and transforms.
- `ui` adds only layout, interaction, themes and automation IDs. A UI element is a twod element, on a fixed layer above the world unless it follows something in it.
- `ui/paint` merges into `twod`, and custom elements draw with twod.

## Alternatives

- Two drawing layers: no migration, but two APIs, two orders and duplicated text and image code.

## Consequences

One drawing pipeline and one order for everything 2D. Hit-testing has to follow an element's transform.

# 0022. A camera names the layers it draws

- Status: Accepted
- Date: 2026-10-30
- Area: rendering
- Owner: architect

## Context

A minimap camera wants markers on a plain background that the main view never draws, and every 2D camera drew every layer of its window.

## Decision

- `twod.Layers(layers…)` makes a camera draw only those layers and what is in them, nested layers included; `Layers()` and `SetLayers` read and change it.
- Without it, a camera draws every layer, as before.
- A window camera's fixed layers and the UI draw whatever it filters: they are the window's, not the world's.

## Alternatives

- **A Hide list, `HideLayers(…)`:** the main view would hide the markers, but every new layer would show in the minimap until hidden there too. Naming what a camera shows fails closed.
- **Bitmask groups,** a `Group(bits)` on layers and a `Mask(bits)` on cameras, as 3D engines do: compact and fast, but it is a second naming scheme for layers that already exist as values, and bit numbers are easy to mix up.
- **A flag per layer, `MinimapOnly()`:** reads well for one minimap, but not for two cameras with different picks.

## Consequences

A camera's picture is described by values the game already holds. Filtering walks an entity's parents for each drawn entity of a filtered camera, which is cheap at 2D scene sizes.

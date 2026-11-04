# 0056. uiv2 images filter in encoded sRGB, as CSS does

- Status: Accepted
- Date: 2026-11-03
- Area: rendering
- Owner: architect

## Context

ember filters textures in linear light: an sRGB view decodes each texel before the sampler blends them. Chrome filters images in encoded sRGB, so a scaled uiv2 picture differed from the page at every edge: midway between greys 59 and 193, 147 here against Chrome's 126.

## Decision

- A texture made with `twod.Encoded()` (`render.Texture.Encoded`) is uploaded as UNORM and filtered as stored; the 2D shader decodes it after filtering, as it already does for a pass's picture.
- uiv2 makes its pictures with it; twod sprites, tiles and every other texture keep linear filtering.
- The web binding takes the same path, as gpu/ is the same.

## Alternatives

- Accept the difference: uiv2 is held to Chrome pixel for pixel, and image edges were its largest remaining gap.
- Filter everything encoded: wrong for game art, which blends in linear light, and every native golden would move.

## Consequences

Like decision 0050, blending follows CSS where uiv2 is compared with Chrome and stays physical for the game; a texture is one or the other, chosen when it is made.

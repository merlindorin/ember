# 0050. The renderer encodes with the exact sRGB curve and blends encoded

- Status: Accepted
- Date: 2026-11-02
- Area: rendering
- Owner: architect

## Context

Colours decode to linear through the exact sRGB curve (`color.Hex`, uiv2's CSS), but the renderer wrote its output with a plain 2.2 power, so dark tones came out lifted: #1e1e1e showed as #232323, and no colour round-tripped. uiv2 is checked against Chrome, which uses the sRGB curve.

## Decision

- Every encode and decode uses the exact piecewise sRGB transfer, as CSS defines it: the 2D shader's output and its pictures read back, the 3D post output, the glow pass, the HDR background and the mip chain (`srgbWGSL`, `color.SRGBToLinear`/`LinearToSRGB`).
- Targets stay UNORM, and blending stays in encoded space, as browsers blend.

## Alternatives

- `*Srgb` surface and pass formats, the GPU encoding after blending: rejected, because it moves all blending into linear light, away from CSS. Anti-aliased text and every translucent layer, light pictures and glows included, would composite differently from Chrome.
- Keep 2.2 and adjust colours on the way in: colours would still not round-trip, and uiv2 would drift from Chrome.

## Consequences

#1e1e1e, #808080 and #f0f0f0 come out byte for byte, and an image drawn 1:1 round-trips. Dark tones are a little darker than before, most visible in unlit ambient (the crypt outside a torch), so the demo's light may want retuning; every golden moved.

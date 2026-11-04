# 0048. A light Affects layers through a picture per set of lights left out

- Status: Accepted
- Date: 2026-11-02
- Area: rendering
- Owner: architect

## Context

`light.Affects(layers…)` makes a light light only some layers. A window has one light picture that every lit drawing reads, so a layer a light leaves out needs a picture without that light.

## Decision

- A layer some light Affects draws, with every layer and entity it holds, nested ones too, in a light class of its own: the chain of such layers it is under, the innermost first, up to 15 a window. A sprite follows its layer.
- For each set of occluder groups left out (ShadedBy) and lights left out (Affects) that drawings need, the window draws one more light picture: a copy of its own with the lights left out taken out, their silhouettes added back, and the lights a left-out group can shade redrawn (decision 0045). A directional light left out is taken out whole.
- The cap is 4 more pictures a window (`draw.MaxPictures`). Past it, drawings fall back to the window's picture, lit by every light. The F3 overlay counts them, and the light system logs the layers once a window, again when they change.
- Static lights light everything.

## Alternatives

- Bit masks, a light's mask matched against a layer's (`Mask`/`Receives`): the same pictures, but the game keeps the bits in step with its layers.
- A light picture per layer, drawn whole: every set redraws every light, the cost ShadedBy had before drawing only the lights reached.
- Affects on single entities: a class per sprite would fill the cap at once; a sprite takes its layer's light, for now.

## Consequences

No light using Affects costs nothing. Each set costs a picture copy and the left-out lights' quads drawn again: two sets of 16 lights of radius 250 over 64 cost about 1.5 ms of GPU. Ember has no debug builds, so the fallback logs in every build, once.

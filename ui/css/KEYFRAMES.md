# @keyframes and animation (parked until M3)

Proposal by ui-developer-2, answered by ui-developer-lead (2026-11-02). Not built: the architect parked it until M3. It is the plan when M3 comes.

## What games write

```css
@keyframes pulse { from { opacity: .6 } 50% { opacity: 1; transform: scale(1.05) } to { opacity: .6 } }
@keyframes slide-in { from { transform: translateX(-40px); opacity: 0 } }   /* missing "to" = the element's own style */
.title  { animation: pulse 2s ease-in-out infinite; }
dialog:open { animation: slide-in 250ms ease-out both; }
```

## Subset

- `@keyframes name { from | to | N% (lists like "0%, 50%") { declarations } }`, with a per-keyframe `animation-timing-function`. A later `@keyframes` of the same name replaces the earlier one.
- Properties in keyframes: the transition set (colours, opacity, transform, box-shadow), plus `visibility` as the one discrete property (it flips at the keyframe, as in Chrome), so menus can show and hide. `display` stays out. Anything else warns with file:line and is dropped; `!important` in a keyframe is ignored with a warning.
- Longhands: `animation-name`, `-duration`, `-timing-function`, `-delay` (negative starts partway), `-iteration-count` (n | infinite), `-direction` (normal | reverse | alternate | alternate-reverse), `-fill-mode` (none | forwards | backwards | both), `-play-state` (running | paused), and the `animation` shorthand with comma lists. Out: `animation-composition`, timelines and ranges, `@keyframes` inside other at-rules.

## Engine

- `css.Transitions` grows animations and is renamed `css.Motion` when this is built (not before). Same `Compute` and `Step(dt)`, same `Shown`; the Document's wiring stays as is.
- Real time only, as transitions: animations run in pause menus. A game that wants game time sets `animation-play-state` from Go.
- Layering as in CSS: Style, then running animations over normal declarations (a longhand that won by `!important` beats the animation; css keeps one bit per longhand), then transitions on top.
- Interpolation reuses transitions' code (premultiplied sRGB, transform lists or Chrome's matrices, shadows, easings). Implicit from/to keyframes take the element's computed value; `var()` and `em` in a keyframe resolve against the element when the animation starts.
- Life cycle, as in Chrome: an animation starts when its name enters the element's `animation-name` list. Changing duration or another longhand updates it in place; removing the name, `display: none`, or removing the node cancels it; adding the name back restarts it.
- Events: `AnimationStarted`, `AnimationIterated`, `AnimationEnded {Node, Name, Elapsed}`, queued by css and drained with `Events()`, re-emitted by the root package as World events.
- Only running animations are stepped; keyframe values are resolved once, at start, into transitions' value type.

## Tests

- GPU-free with a fake dt: iterations, alternate, fill modes, negative delay, play-state, restart on re-add, events, `!important` over an animation.
- Chrome: pause `document.getAnimations()` at several `currentTime`s per case, as the transitions case does.

Estimate: about a day, mostly the life cycle and its tests.

# Debugging and performance

![The light picture, the occluders outlined, and the shadow maps, over the same crypt](debug-views.png "ember · light")

| Key, with F3 shown | Call | Shows |
|---|---|---|
| F5 | `light.SetDebug(win, light.DebugLight)` | the light each spot gets |
| F5 again | `light.DebugOccluders` | what blocks light: red group 0, green 1, blue 2 |
| F5 again | `light.DebugShadows` | a band per light, −180° at the left: purple near, yellow far, teal open |
| F6 | `light.SetOutlines(win, true)` | every occluder outlined |

```go
if in.Pressed(input.KeyO) {
	light.SetDebug(win, light.DebugOccluders)
	light.SetOutlines(win, true)
	light.SetDebugArea(win, pane) // a rect of the window; empty is all of it
}
```

F3 also shows "lights 1 (1 cast) of 256, 0 static, budget 3 (1 fading), pictures 1 of 4  occluders 21", where pictures counts those of `ShadedBy` and `Affects` and how many layers fell back past four, and a "gpu frame" row.

## No shadow?

| Check | Fix |
|---|---|
| nothing in the occluder view | `light.Occluder()` on the sprite; for tiles, `tiles.Solid` or collision shapes, the map in the window, and no `light.NoShadows()` on it |
| the tile shows, but no shadow | its `shadow` property is `none` |
| shadows from some things only | the light's `BlockedBy` groups |

## Static lights

![A live lamp: the occluding crate casts a shadow; the same lamp Static: the column and wall still shadow, the crate doesn't](static.png "ember · light")

```go
torch := light.NewPoint(150, warm, light.At(x, y), light.Static()) // point or cone lights
torch.SetStatic(false)
```

| | Live | Static |
|---|---|---|
| cost each frame | one light | about nothing |
| redrawn | every frame | where it or the space's tiles change |
| shadows from tile maps and solids | yes | yes |
| shadows from bodies and `Occluder` sprites | yes | no |
| counts toward the 256 lights and shadow rows | yes | no |

## Budget and priority

![Five lamps along the floor; with the camera's Budget(3), the two outer ones are put out](budget.png "ember · light")

```go
camera := twod.NewCamera(light.Budget(3), light.BudgetFade(0.3)) // at most 3 drawn, 0 for no cap
boss := light.NewPoint(200, red, light.Priority(10))             // kept before the others
light.SetBudget(camera, 8)
```

| Past the budget | |
|---|---|
| kept first | higher `Priority`, then the disc nearest the middle of the view |
| a light already kept | ranks as if 10% nearer, so lights about as near never swap back and forth |
| a light dropped | fades out over `BudgetFade` seconds, 0.3 by default, and back in when it makes it again; a new light shows at once |
| directional and `Static` lights | count against no budget |

Fades run on the World's scaled time, so a paused game holds them, and a fading light still draws: a view can hold a few more than its budget for a moment ([decision 0047](../../99-decisions/0047-lights-past-a-camera-budget-fade-out-by-priority-then-distance.md)).

## Limits

| A window takes | |
|---|---|
| lights | 256 |
| shadow rows | 256, a shape light with shadows takes 8 |
| directional lights | 4 |
| occluder groups | 3 |
| probes | 1,024 |

A camera drawing `Into` a render texture draws lit layers unlit. A light without Elevation, Size or a fade costs what it did before they existed, and a light with no occluder of its groups within its radius (box `Occluder` sprites, solids, tile shadows; silhouettes don't count) costs no shadow work at all, as with `Shadows(false)`.

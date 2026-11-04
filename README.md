<div align="center">

# ember

**A 2D game engine in plain Go. WebGPU on the desktop and in the browser.**

[![CI](https://github.com/merlindorin/ember/actions/workflows/ci.yml/badge.svg)](https://github.com/merlindorin/ember/actions/workflows/ci.yml)
[![Go 1.27](https://img.shields.io/badge/go-1.27-00ADD8?logo=go&logoColor=white)](go.mod)
[![MIT](https://img.shields.io/badge/licence-MIT-green)](LICENSE)
[![Guide](https://img.shields.io/badge/guide-ember.vanyda.cloud-orange)](https://ember.vanyda.cloud/ember/)

![A horde closing in on the hero at a crypt door, lit by torches, in Lantervale](.github/readme/horde.png)

<sub>**Lantervale**, our game in development, made with ember.</sub>

</div>

## Highlights

- 🧱 **Plain Go ECS**: entities are structs, systems are small objects. No codegen, no editor.
- 🔥 **WebGPU renderer**: Metal, Vulkan, Direct3D 12, and the browser with the same code.
- 💡 **2D lights**: point, cone, directional and shape lights, soft shadows.
- ✨ **Effects**: particles, WGSL shaders, blend modes, render textures.
- 🗺️ **Worlds**: sprites, animations, tile maps, Tiled levels, collision, pathfinding.
- 🖼️ **UI in HTML + CSS**: flex, grid, the cascade, checked pixel for pixel against Chrome.
- 🔤 **Text**: font files, bitmap fonts, colour emoji, `:shortcodes:`.
- 🎮 **Input and sound**: keyboard, mouse, gamepads, touch, gestures; music, positional sound, reverb.
- 🧪 **Headless tests**: no GPU, no real time, every UI element reachable by path.

## Samples

<table>
<tr>
<td width="50%"><a href="docs/06-2d/19-lights/02-point-lights.md"><img src=".github/readme/point-lights.gif" alt="A torch casting soft shadows from trees and pillars"></a></td>
<td width="50%"><a href="docs/06-2d/19-lights/04-cone-lights.md"><img src=".github/readme/cone-light.gif" alt="A flashlight cone aimed at the mouse"></a></td>
</tr>
<tr>
<td align="center"><sub>Point lights and soft shadows</sub></td>
<td align="center"><sub>Cone lights</sub></td>
</tr>
<tr>
<td><a href="docs/06-2d/17-particles.md"><img src=".github/readme/particles.gif" alt="A campfire of flames, sparks and smoke"></a></td>
<td><a href="docs/06-2d/16-2d-shader.md"><img src=".github/readme/shader.gif" alt="Ripples from a WGSL shader where the mouse clicks"></a></td>
</tr>
<tr>
<td align="center"><sub>Particles</sub></td>
<td align="center"><sub>WGSL shaders</sub></td>
</tr>
<tr>
<td><a href="docs/05-ui/09-walkthroughs/03-controls.md"><img src=".github/readme/ui-controls.gif" alt="A settings panel: text input, slider, progress bar, checkboxes"></a></td>
<td><a href="docs/07-text/03-emoji.md"><img src=".github/readme/emoji.png" alt="Monochrome and colour emoji in text and a chat input"></a></td>
</tr>
<tr>
<td align="center"><sub>UI controls</sub></td>
<td align="center"><sub>Colour emoji</sub></td>
</tr>
<tr>
<td><img src=".github/readme/sanctuary.png" alt="Lantervale's sanctuary lit by torches and candles"></td>
<td><img src=".github/readme/power-ups.png" alt="Lantervale's power-up cards over the paused game"></td>
</tr>
<tr>
<td align="center"><sub>Lantervale: a lit room</sub></td>
<td align="center"><sub>Lantervale: a UI screen</sub></td>
</tr>
</table>

## Quick start

Needs Go 1.27, [Task](https://taskfile.dev) (`brew install go-task`) and a [WebGPU-capable GPU](docs/01-getting-started/README.md).

```sh
git clone https://github.com/merlindorin/ember
cd ember
task wgpu                        # the WebGPU library
go run ./examples/01-hello-cube
```

## Hello

```go
package main

import (
	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func main() {
	win := platform.NewWindow(platform.Title("hello"), platform.Size(640, 360))
	win.Add(twod.NewText("Hello, ember", twod.At(320, 180), twod.FontSize(32)))
	ember.New().AddEntities(win).MustRun()
}
```

## Entities and systems

```go
package main

import (
	"fmt"

	"github.com/merlindorin/ember/world"
)

type Heat struct{ Degrees float32 }

func (h *Heat) GetHeat() *Heat { return h }

type Flame struct {
	*world.BasicEntity
	Heat
	Name string
}

type Burning interface {
	world.BasicFace
	GetHeat() *Heat
}

type Cooling struct{ world.Group[Burning] } // receives every entity that is Burning

func (s *Cooling) Update(f world.Frame) {
	for flame := range s.All() {
		flame.GetHeat().Degrees -= f.DT
		if flame.GetHeat().Degrees <= 0 {
			fmt.Printf("%.1fs  %s went out\n", f.Elapsed, flame.(*Flame).Name)
			f.World.RemoveEntity(flame)
		}
	}
}

func main() {
	world.New().
		AddEntities(
			&Flame{BasicEntity: world.NewBasic(), Heat: Heat{2}, Name: "candle"},
			&Flame{BasicEntity: world.NewBasic(), Heat: Heat{5}, Name: "campfire"},
		).
		AddSystems(&Cooling{}).
		On(world.Update, "quit", func(f world.Frame) {
			if f.World.Len() == 0 {
				f.Quit()
			}
		}).
		MustRun()
}
```

## Learn

- 📖 [The guide](https://ember.vanyda.cloud/ember/): one idea per page, its examples runnable in the page (`task docs` serves it locally)
- 🧩 [56 examples](examples): `go run ./examples/NN-name`
- 🧭 [Design decisions](docs/99-decisions/README.md): each choice and the alternatives it beat

> ember is young: its API still moves, and the guide shows only what is ready.

## Licence

Code under [MIT](LICENSE). Pictures, fonts and sounds: see [CREDITS.md](CREDITS.md).

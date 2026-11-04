# ember

ember is a Go game engine built on its own ECS: entities are plain structs, systems are small objects, and the same simulation runs on a headless server and a windowed client. It draws 2D games with WebGPU, sprites, tile maps, lights and shadows, particles and text, has a UI of HTML elements styled by a CSS subset, plays sound, and runs its examples in the browser. A full game built with ember is in development.

**The guide:** [ember.vanyda.cloud/ember](https://ember.vanyda.cloud/ember/), every page a small example you can run in the page.

## Requirements

- Go 1.27
- [Task](https://taskfile.dev) (`brew install go-task`)
- A graphics card WebGPU draws with: [Getting started](docs/01-getting-started/README.md) lists them

## Install

```sh
git clone https://github.com/merlindorin/ember
cd ember
task wgpu                          # fetches the WebGPU library the renderer loads
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

The [guide](https://ember.vanyda.cloud/ember/) goes on from here, one example at a time; `task docs` serves it locally.

## Licence

ember's code is under the [MIT licence](LICENSE). Its pictures, fonts and sounds come with their own licences, listed in [CREDITS.md](CREDITS.md) and in the CREDITS files beside them.

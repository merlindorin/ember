# Explicit services

Status: approved (decision 0015); being implemented.

`world.Resource[T]` stays for a game's own shared values, such as `Weather`
in 03-resources. The engine stops exposing its own services through it,
because a call like `world.Resource[platform.Window](f.World)` hides the
dependency, uses the type as the key, and fails at runtime. Instead, a game
creates the engine's services as values, with constructors and options,
hands them to `ember.New` and to its own systems, and holds them.

## Principles

- **The World's own state is World API.** `f.Time` on the Frame, and `w.Time()` and `w.Stats()` on the World.
- **Services are values.** They are made with constructors and functional options (`platform.NewWindow`, `render.NewEnvironment`, `net.NewClient`…). A game passes them to `ember.New` (`ember.Automation(auto)`) or adds them as entities, and keeps them for its systems. When none is passed, `ember.New` makes a default one.
- **Windows are entities.** `platform.NewWindow` returns a `*platform.Window` entity, and the first window added to the World is the primary one.
- **Plugins share state privately.** Engine plugins talk through unexported fields and the values `ember.New` wires between them, never through public resources.
- **A guard test.** A test fails if a package outside `world` calls `SetResource`, `Resource`, `TryResource` or `ResourceOrInit` with an engine type.

## Inventory and replacements

| Today (resource)                                 | Read by                          | Becomes                                                                                                                                         |
|--------------------------------------------------|----------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|
| `world.Time`                                     | games, automation, ui            | `f.Time` on the Frame and `w.Time()`, which also has `SetScale` and `Pause`                                                                     |
| `world.Stats`                                    | editor, paint overlay            | `w.Stats()`                                                                                                                                     |
| `platform.Window` (primary)                      | games, ui, gpu, desktop          | the first `*platform.Window` entity; games hold it                                                                                              |
| `platform.Current`                               | gpu, gui                         | internal to the presenters                                                                                                                      |
| `input.State`                                    | games, camera, paint, editor     | `win.Input()`. The primary window's input also holds the gamepads                                                                               |
| `render.Environment`                             | games, node3d, editor            | an entity: `render.NewEnvironment(opts…)` added to the World, the first one lighting the scene; `node3d.Environment` is one; `ember.New` adds a default if none |
| `render.Post`                                    | 90-pixel-isle, gpu               | a camera's: `render.Camera{Post: render.NewPost(opts…)}`                                                                                        |
| `render.View`, `render.Views`                    | games (picking, HUD), ui.Follow  | `cam.View()` on the camera entity; ui.Follow reads its window's camera internally                                                               |
| `render.Gizmos`, `render.Billboards`             | games, debug drawing             | `render.NewGizmos()` and `render.NewBillboards()` entities that the renderer draws                                                              |
| `net.Server`, `net.Client`                       | Emberfall, nettest, node3d       | `net.NewServer(opts…)` and `net.NewClient(conn, opts…)`, which are plugins the game adds and holds                                              |
| `automation.Registry`, `automation.Server`       | games (actions), ui, gui, editor | `auto := automation.New(opts…)` passed with `ember.Automation(auto)`; games call `auto.Define`, and ui and gui record through it internally |
| `editor.Editor`                                  | games (panels), studio           | `ed := editor.New(opts…)` passed with `ember.WithEditor(ed)`; games call `ed.AddPanel`                                                          |
| `paint.UI`, `paint.Canvas`, `paint.DebugOverlay` | ui system, gpu, editor           | internal to the ui plugin. Custom elements get `*paint.UI` in `Draw`, and overlays are custom elements                                          |
| `gui.GUI`                                        | editor, studio                   | `gui.New(opts…)`, held by the editor and studio                                                                                                 |
| `node.Tree`                                      | games, canvas, SceneService      | `tree := node.New(w)` as today, held by the game; `automation.Scenes(tree)` serves it                                                       |
| `canvas.Settings`                                | 2D games                         | tree options: `node.New(w, canvas.SnapTransforms(), canvas.SnapVertices())`                                                                     |
| `ui.Theme`, `ui.TranslationServer`               | ui                               | gone with the old ember/ui ([0059](../99-decisions/0059-ember-ui-is-uiv2.md)): each ui Document takes `kit.Theme()` and `ui.WithTranslator(tr)` |
| `spatial.Index`                                  | games                            | `idx := spatial.NewIndex(cell)` and `spatial.NewIndexer(idx)`, held by the game                                                                 |
| `gpu.Renderer`                                   | gpu, desktop                     | internal                                                                                                                                        |
| event queues                                     | `world`                          | stay internal to `world`, which is fine since it is the World's own storage                                                                     |

`ember.WithTitle`, `WithSize`, `WithPosition`, `WithVSync` and
`WithBackground` go away: those are the window's options. `ember.New` with
no window entity opens a default one, titled "ember", 1280×720.

## Before and after

`09-window`:

```go
// before
ember.New(ember.WithTitle("ember · window"), ember.WithSize(640, 360), ember.WithBackground(color.Hex("#2b2f3a"))).
	AddEntities(size).
	AddSystems(&ShowSize{label: size}).
	MustRun()

func (s *ShowSize) Update(f world.Frame) {
	size := world.Resource[platform.Window](f.World).Size()
	…
}

// after
win := platform.NewWindow(platform.Title("ember · window"), platform.Size(640, 360), platform.Background(color.Hex("#2b2f3a")))
size := ui.NewLabel("", ui.At(16, 16))
win.AppendChild(size)

ember.New().
	AddEntities(win).
	AddSystems(&ShowSize{window: win, label: size}).
	MustRun()

func (s *ShowSize) Update(world.Frame) {
	size := s.window.Size()
	…
}
```

`10-multi-window`: the `Window` type in `window.go` goes away, and the
scene window is an entity like the map's.

```go
main := platform.NewWindow(platform.Title("ember · scene"), platform.Size(900, 600), platform.Position(60, 80))
main.AppendChild(ui.NewLabel("scene", ui.At(16, 16)))

ember.New().
	AddEntities(
		main,
		newGround(), newCrate(-2, 0), newCrate(2, 1),
		&Eye{world.NewBasic(), scene.At(6, 5, 8).LookingAt(geom.Zero3), render.Camera{}},
		newMapWindow(),
	).
	MustRun()

func newMapWindow() *platform.Window {
	w := platform.NewWindow(platform.Title("ember · map"), platform.Size(400, 400), platform.Position(980, 80))
	w.AppendChild(&Eye{world.NewBasic(), scene.At(0, 20, 0.01).LookingAt(geom.Zero3), render.Camera{OrthoHeight: 12, Window: w}})
	w.AppendChild(ui.NewLabel("map", ui.At(16, 16)))
	return w
}
```

A system that reads input (90-pixel-isle's `Controls`):

```go
// before
func (c *Controls) Update(f world.Frame) {
	in := world.Resource[input.State](c.world)
	…
}

// after
type Controls struct {
	input *input.State
	…
}

controls := &Controls{input: win.Input(), game: g}

func (c *Controls) Update(f world.Frame) {
	stick := c.input.Vector(input.KeyA, input.KeyD, input.KeyW, input.KeyS)
	…
}
```

Emberfall's network client:

```go
// before
a.AddPlugins(net.ClientPlugin{Conn: opts.Conn, Codec: game.ClientCodec, TickRate: game.TickRate, Hello: hello})
s := world.SetResource(a, &Session{Client: world.Resource[net.Client](a), …})

// after
client := net.NewClient(opts.Conn, net.WithCodec(game.ClientCodec), net.TickRate(game.TickRate), net.Hello(hello))
a.AddPlugins(client)
s := &Session{Client: client, …}
```

The Session itself is game state, so it may stay a game resource or become
a value its systems hold. The latter matches the examples.

## Migration, each step green

1. **The World's own state.** Add `f.Time`, `w.Time()` and `w.Stats()`, and drop `world.Time` and `world.Stats` as resources. Automation reads time as `time/scale`.
2. **Windows.**
   - `platform.NewWindow` returns a `*platform.Window` entity, and the first one added is the primary.
   - Remove `ember.WithTitle` and the other window options from `ember`.
   - Input is `win.Input()`; `input.State` and `platform.Window` stop being resources, and `platform.Current` becomes internal.
   - Rewrite 08 and 09, and every `world.Resource[input.State]`.
3. **Render.** The Environment as an entity, `Camera.Post`, `cam.View()`, and Gizmos and Billboards as entities. Update 90-pixel-isle, 91-character, 92-sword, the editor's lighting panel and Emberfall.
4. **Net.** `net.NewServer` and `net.NewClient`, with Emberfall, nettest and node3d's replicator holding them.
5. **Tools.** `automation.New` and `editor.New` with `ember.Automation` and `ember.WithEditor`; the theme and translator through `ember`; paint, gui and gpu become internal. Studio goes through game-supporting-tools-engineers.
6. **Scenes and the rest.** `node.Tree` held by the game, served with `automation.Scenes`; canvas settings become tree options; `spatial.NewIndex`.
7. **The guard test** and the README, guide and CLAUDE.md pass.

**What goes away:** every engine type from the inventory as a resource;
`ember.WithTitle`, `WithSize`, `WithPosition`, `WithVSync` and
`WithBackground`; the plugins that only set resources (`net.ServerPlugin`,
`net.ClientPlugin` and `automation.Plugin`, which become the values
themselves); and `platform.Resolve(w, nil)`, since the primary window is a
value the game holds.

## Decided

1. Engine state gets its own automation paths (`time/scale`, `window/size`…); `resource/…` stays for game resources.
2. The environment is an entity, like windows; the scene's `node3d.Environment` is simply one.
3. Post belongs to the camera.
4. Gamepads are read through the primary window's input.
5. No `ember.Server` helper for now.

# Input and sound

Packages `github.com/merlindorin/ember/input` and `github.com/merlindorin/ember/audio`. Guide: `docs/08-input`, `docs/09-audio`. Examples: `examples/47-keyboard` … `examples/55-audio-effects`.

## Where input comes from

- Each window has its own: `(w *platform.Window) Input() *input.State`. Keep it in your system: `&Play{in: win.Input()}`.
- Read it in `PreUpdate` (then FixedUpdate uses it the same frame) or `Update`.
- `input.Key`, `input.MouseButton` and `input.GamepadButton` are all `input.Button`: every read below takes any of them.
- A `Key` is a physical **position** (`KeyW` is the key there on a French AZERTY too); typed characters come from `Text()`.

## Read a key or a button

| Want | Call |
|---|---|
| held now | `(s *State) Down(b Button) bool` |
| went down this frame (once per press) | `Pressed(b Button) bool` |
| went up this frame | `Released(b Button) bool` |
| analog level 0..1 (trigger, stick direction) | `Value(b Button) float32` |
| -1, 0, 1 from two buttons | `Axis(neg, pos Button) float32` |
| a move vector, **Y down** (up is negative), length ≤ 1 | `Direction(left, right, up, down Button) geom.Vec2` |

Names: `input.KeyA`…`KeyZ`, `Key0`…`Key9`, `KeyLeft/Right/Up/Down`, `KeySpace`, `KeyEnter`, `KeyEscape`, `KeyBackspace`, `KeyTab`, `KeyShift`, `KeyControl`, `KeySuper`.
Mouse: `input.MouseLeft`, `MouseRight`, `MouseMiddle`, `MouseBack`, `MouseForward`.

```go
move := in.Direction(input.KeyA, input.KeyD, input.KeyW, input.KeyS)
hero.Translate(move.Scale(160 * f.DT))
if in.Pressed(input.KeySpace) {
	jump()
}
```

Gotchas:
- A tap shorter than a frame still shows as a press then a release.
- While a ui element has focus, the keys it uses read as up: `Captured(b) bool`, `KeyboardCaptured() bool`, `PointerCaptured() bool` (pointer over the ui).
- `LastDevice() input.Device` (`Keyboard`, `Mouse`, `Gamepad`, `TouchDevice`) picks which button hints to show.

## Typed text and the clipboard

- `Text() []rune`: the characters typed this frame, in the player's layout. Append them; handle `KeyBackspace` with `Pressed`.
- `(w *Window) Clipboard() string` / `SetClipboard(text string)`. Paste on `(Down(KeyControl) || Down(KeySuper)) && Pressed(KeyV)`.
- For real text fields use a ui `Input`/`TextArea` instead (selection, undo): see ui.md. Example: `examples/47-keyboard`.

## The mouse

| Want | Call |
|---|---|
| pointer in window points | `Mouse() geom.Vec2` |
| movement since last frame | `MouseDelta() geom.Vec2` |
| wheel this frame, in lines (**Y up is negative**) | `Wheel() geom.Vec2` |
| files dropped on the window | `Dropped() []string` |
| pointer in the world | `cam.ScreenToWorld(in.Mouse())` on the `*twod.Camera` |
| cursor shape | `win.SetCursor(platform.CursorArrow / CursorHand / CursorText / CursorHidden)` |

Example: `examples/48-mouse` (drag crates, wheel zoom, back/forward).

## Intent instead of keys: an action map

Bind an action type to buttons once; read actions, never keys. Keys, mouse and pads mix in one binding. Example: `examples/49-actions`.

- `input.Bind[A comparable](action A, buttons ...Button) input.Binding[A]`
- `input.NewActions[A comparable](binds ...Binding[A]) *input.Actions[A]`
- Reads take an `input.Reader` (the `*input.State`, or one `*input.Gamepad` for a given player): `Down(r, a)`, `Pressed(r, a)`, `Released(r, a)`, `Value(r, a)`, `Axis(r, neg, pos)`, `Direction(r, left, right, up, down)`.
- Rebind: `(a *Actions[A]) Bind(action A, buttons ...Button)`; `Bindings(action) []Button`; `Bound(b) (A, bool)`.
- Save/load: `json.Marshal(actions)` / `json.Unmarshal` (actions are named by `fmt.Sprint(action)`: give the type a `String()`).

```go
type Act int

const (
	Left Act = iota
	Right
	Up
	Down
	Fire
)

func (a Act) String() string { return [...]string{"left", "right", "up", "down", "fire"}[a] }

controls := input.NewActions(
	input.Bind(Left, input.KeyLeft, input.KeyA, input.GamepadLeftStickLeft),
	input.Bind(Right, input.KeyRight, input.KeyD, input.GamepadLeftStickRight),
	input.Bind(Up, input.KeyUp, input.KeyW, input.GamepadLeftStickUp),
	input.Bind(Down, input.KeyDown, input.KeyS, input.GamepadLeftStickDown),
	input.Bind(Fire, input.MouseLeft, input.KeySpace, input.GamepadSouth),
)
dir := controls.Direction(in, Left, Right, Up, Down)
```

## Gamepads

On the **primary** window's input. Example: `examples/50-gamepads`.

- Buttons by position: `GamepadSouth/East/West/North`, `GamepadUp/Down/Left/Right` (cross), `GamepadLeftShoulder/RightShoulder`, `GamepadStart`, `GamepadBack`, `GamepadGuide`, `GamepadLeftStickPress`….
- Analog (read with `Value`, or `Down` past 0.5): `GamepadLeftTrigger`, `GamepadRightTrigger`, `GamepadLeftStickLeft/Right/Up/Down`, `GamepadRightStick…`.
- `Down/Pressed` on the State read **any** connected pad. Per player: `in.Gamepads() []*input.Gamepad`, `in.Gamepad(id int) *input.Gamepad`.
- `in.Stick(input.LeftStick) geom.Vec2` (or `input.RightStick`), `SetDeadzone(r float32)`.
- World events: `input.GamepadConnected{ID, Name}`, `input.GamepadDisconnected`.

## Touch and gestures (macOS for now)

- `in.Touches() []input.Touch` (`ID`, `Source`, `At`, `Phase`). A `Trackpad` touch is 0–1 on the pad, never window points; a `Touchscreen` one is in points.
- `Pinch() float32`, `Rotation() float32`, `Pan() geom.Vec2`, `Swipe() geom.Vec2` this frame.
- Events: `input.TouchStarted`, `TouchMoved`, `TouchEnded` (each `{Touch}`). Examples: `examples/51-touch`, `examples/52-gestures`.

## Simulated input in tests

Full recipes in [testing.md](testing.md). In short: set input, then run frames. What you set shows from the next frame.

```go
in := win.Input()
in.SetKey(input.KeyD, true)
w.RunFrames(60)
in.SetKey(input.KeyD, false)
w.RunFrames(1)
in.SetMouse(geom.V2(420, 100))
in.SetMouseButton(input.MouseLeft, true)
in.AddWheel(geom.V2(0, -5))
in.AddText('a')
in.ConnectGamepad(0, "pad")
in.SetGamepadStick(0, input.LeftStick, geom.V2(-1, 0))
```

Also `SetGamepadButton(id, b, down)`, `SetGamepadTrigger(id, trigger, pull)`, `AddTouch(id, source, at)`, `MoveTouch`, `EndTouch`, `AddPinch`, `AddDrop(paths...)`.

## Complete program: walk with an action map

```go
package main

import (
	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

type Act int

const (
	Left Act = iota
	Right
	Up
	Down
	Reset
)

type Walk struct {
	hero     *twod.Text
	in       *input.State
	controls *input.Actions[Act]
}

func (w *Walk) Phase() world.Phase { return world.PreUpdate }

func (w *Walk) Update(f world.Frame) {
	w.hero.Translate(w.controls.Direction(w.in, Left, Right, Up, Down).Scale(200 * f.DT))
	if w.controls.Pressed(w.in, Reset) {
		w.hero.SetPosition(geom.V2(320, 180))
	}
}

func main() {
	win := platform.NewWindow(platform.Title("walk"), platform.Size(640, 360), platform.FixedSize())
	hero := twod.NewText("@", twod.At(320, 180), twod.FontSize(32))
	win.Add(hero)
	controls := input.NewActions(
		input.Bind(Left, input.KeyLeft, input.KeyA, input.GamepadLeftStickLeft),
		input.Bind(Right, input.KeyRight, input.KeyD, input.GamepadLeftStickRight),
		input.Bind(Up, input.KeyUp, input.KeyW, input.GamepadLeftStickUp),
		input.Bind(Down, input.KeyDown, input.KeyS, input.GamepadLeftStickDown),
		input.Bind(Reset, input.KeySpace, input.GamepadSouth),
	)
	ember.New().AddEntities(win).AddSystems(&Walk{hero: hero, in: win.Input(), controls: controls}).MustRun()
}
```

## Sound: the mixer

The mixer is an **entity**: `ember.New().AddEntities(win, mixer)`. The device opens only when a mixer is added, and never under `EMBER_HIDDEN`; without a device it mixes at the World's pace.

- `audio.NewMixer(opts ...audio.MixerOption) *audio.Mixer`; options `audio.WithBus(name string, volume float32, opts ...Option)`, `audio.MaxVoices(n)`, `audio.SampleRate(hz)`.
- Formats: import `_ "github.com/merlindorin/ember/audio/wav"` and/or `_ "github.com/merlindorin/ember/audio/vorbis"` (Ogg).
- Short sounds, decoded whole: `audio.Load(fsys fs.FS, name string) (*audio.Clip, error)`. Made in code: `audio.NewClip(rate, channels int, samples []float32)`.
- Music and long loops, streamed: `audio.Open(fsys fs.FS, name string) (*audio.Stream, error)`.

## Play a sound

- `(m *Mixer) Play(s Sound, opts ...Option) *Voice`: Clip or Stream.
- Options: `audio.Volume(v)` (keep ≤ 1), `audio.Pan(p)` (-1 left … 1 right), `audio.Pitch(p)` (1 normal), `audio.Loop()`, `audio.OnBus(name)`, `audio.FadeIn(d)`, `audio.FadeOut(d)`, `audio.Priority(n)`.
- A Voice: `Stop(opts...)`, `Pause()`, `Resume()`, `Playing() bool`, `SetVolume/SetPan/SetPitch`. `m.Voices() int` counts what plays.
- When a voice ends the World gets `audio.VoiceFinished{Voice, Stopped}`.
- Example: `examples/53-play-a-sound` (random pitch, pan by click).

## Music

- `(m *Mixer) PlayMusic(s Sound, opts ...Option) *Voice`: one music voice at a time, on bus "music" unless `OnBus` says otherwise (make it with `WithBus("music", v)` to set its volume). `audio.Crossfade(d)` from the current track; `m.Music() *Voice`; `m.StopMusic(opts...)`.
- Buses: `m.Bus(name) *Bus` with `SetVolume`, `SetMuted`; `m.Master()`. Example: `examples/54-music-and-positional`.

```go
mixer := audio.NewMixer(audio.WithBus("music", 0.5), audio.WithBus("sfx", 1))
mixer.PlayMusic(day, audio.Loop(), audio.FadeIn(time.Second))
mixer.PlayMusic(night, audio.Loop(), audio.Crossfade(2*time.Second)) // later
```

## Sounds in the world: emitters and a listener

- A component `*audio.Emitter` on a twod entity sounds where the entity is: `audio.NewEmitter(sound Sound, opts ...Option)`, with `audio.Distance(min, max)`, `audio.Rolloff(r)`, `audio.Autoplay()` plus the voice options. `Play()`, `Stop()`.
- A component `audio.Listener` (on the hero or the camera) hears them; the first in the World is used. In 2D it faces up at rotation 0.

```go
type Campfire struct {
	twod.Item
	*audio.Emitter
}

fire := twod.Build(&Campfire{Item: twod.Base(), Emitter: audio.NewEmitter(crackle,
	audio.Loop(), audio.Autoplay(), audio.OnBus("sfx"), audio.Distance(30, 500))}, "campfire", twod.At(90, 220))
```

## Effects

- `audio.NewReverb(audio.RoomSize(v), audio.Damping(v), audio.Wet(v))`, `audio.NewLowPass(audio.Cutoff(hz), audio.Resonance(q))`, `audio.NewHighPass(...)`, `audio.NewEcho(audio.Delay(d), audio.Feedback(v), audio.Wet(v))`.
- Chain with `audio.Effects(e...)` on a voice/emitter, or a bus: `audio.WithBus("sfx", 1, audio.Effects(room))`, `bus.SetEffects(...)`.
- Setters (`SetCutoff`, `SetRoomSize`, `SetWet`…) glide over 10 ms: tween them freely. **One effect value belongs to one bus or voice.** Example: `examples/55-audio-effects`.

## Sound in tests

`audio.NewRecorder(m *Mixer) *audio.Recorder` records what the mixer outputs as the World runs: `Frames() int`, `Peak(from, to int) float32`, `PeakLR(from, to) (l, r float32)`, `Reset()`. An emitter's `Heard() (gain, pan float32)` is what the listener hears.

## Complete program: a beep made in code

```go
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

type Beeper struct {
	in    *input.State
	mixer *audio.Mixer
	beep  *audio.Clip
}

func (b *Beeper) Update(world.Frame) {
	if b.in.Pressed(input.KeySpace) {
		b.mixer.Play(b.beep, audio.Pitch(0.8+0.4*rand.Float32()), audio.Volume(0.5)) //nolint:gosec // a game's randomness
	}
}

func main() {
	const rate = 44100
	samples := make([]float32, rate/5)
	for i := range samples {
		fade := 1 - float32(i)/float32(len(samples))
		samples[i] = fade * float32(math.Sin(2*math.Pi*440*float64(i)/rate))
	}
	beep, err := audio.NewClip(rate, 1, samples)
	if err != nil {
		log.Fatal(err)
	}
	mixer := audio.NewMixer()
	win := platform.NewWindow(platform.Title("beep"), platform.Size(640, 360), platform.FixedSize())
	win.Add(twod.NewText("Space beeps", twod.At(320, 180), twod.FontSize(24)))
	ember.New().
		AddEntities(win, mixer).
		AddSystems(&Beeper{in: win.Input(), mixer: mixer, beep: beep}).
		MustRun()
}
```

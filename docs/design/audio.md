# Audio (design draft)

Status: built in `ember/audio` (decision [0020](../99-decisions/0020-sound-is-pure-go-behind-one-device.md)), with examples 45-play-a-sound and 46-music-and-positional. As built: no MP3; events are `VoiceFinished` and `VoiceLooped`; `MaxVoices(n)`; an emitter's `Heard()` gives the gain and pan last heard; the open questions below were answered oto, no MP3, automatic device, goroutine decoding (synchronous without a device), orientation from the transform, no networked sound. Ember has no sound; raylib has `LoadSound`/`PlaySound`, music streams, volume, pitch and pan. This design covers that plus buses, crossfades and positional sound, in ember's style: values and entities built with options (0007), no resources (0015), pure Go everywhere a server or a test runs (0003, 0004, 0009).

## Packages

| Package | cgo | Holds |
|---|---|---|
| `ember/audio` | no | `Clip`, `Stream`, the `Mixer`, `Voice`, buses, `Listener`/`Emitter` components, the positional system, events, a format registry (`audio.RegisterFormat`, like `image`) |
| `ember/audio/wav`, `audio/vorbis`, `audio/mp3` | no | decoders, registered by a blank import, each wrapping one library |
| `ember/internal/audiodevice` | per OS | the desktop output, opened by `platform/desktop`'s runner when the game adds a `Mixer` |

Servers and `go test` never import `audiodevice`: with no device the mixer runs against a null output, and tests use `audio.Recorder`.

## Decoding in pure Go

- **WAV**: our own reader (PCM 8/16/24-bit, 32-bit float, mono/stereo), about 150 lines, fuzzed like `tga`.
- **Ogg Vorbis**: `github.com/jfreymuth/oggvorbis` v1.0.3 (+ `jfreymuth/vorbis` v1.0.2): pure Go, `go 1.15`, no other deps, already in the module cache.
- **MP3 (optional)**: `github.com/hajimehoshi/go-mp3`. The cached v0.3.2 pulls the old `oto` v0.6.1 (and an old x/mobile) into the module graph; v0.3.4 dropped that dependency but isn't cached. Take v0.3.4 or skip MP3.
- Every decoder yields interleaved float32 frames at its own rate. The mixer resamples to its output rate (linear, the same resampler as pitch).
- `audio.Load(fsys, name)` decodes a whole `*Clip` (short sounds, kept in memory). `audio.Open(fsys, name)` returns a `*Stream` that decodes as it plays (music), seekable for loops.

## Output

- **Desktop**: `github.com/ebitengine/oto/v3` v3.3.2: `go 1.18`, requires `ebitengine/purego` v0.8.0 (cached) and `golang.org/x/sys` v0.25.0 (below our v0.39, so nothing moves up).
  - macOS: CoreAudio through purego, **no cgo**.
  - Windows: WASAPI, no cgo.
  - Linux: ALSA, needs cgo and `libasound2-dev` (the desktop build already uses cgo for GLFW).
  - Only its `go.mod` is in the cache, so adding it takes one network fetch, which I'd do with `GOFLAGS=-mod=mod` and check that `go.sum` moves nothing else.
- The device thread pulls float32 frames from the mixer (`Mixer.Read(buf)`); the 48 kHz stereo default can be changed with options. Latency target about 20–40 ms (oto buffer size option).
- **Headless**: `audio.Recorder` drives the mixer by frames instead of a clock, and keeps the mixed samples (or just peaks and per-voice positions) so tests assert what was heard, with no device and no real time.

## Mixer

- One `*audio.Mixer` per game, an entity (`audio.NewMixer(opts...)` added with `AddEntities`). Its `System` advances voices, applies commands and emits events on the main loop. The device thread only reads a lock-protected snapshot of voice parameters and pulls samples, so game code never races the audio thread.
- **Voices**: `mixer.Play(clip, opts...) *Voice` plays a one-shot; any number at once, with a cap (`WithMaxVoices(64)`, the oldest of the lowest priority is stolen). A `*Voice` has `Stop`, `Pause`, `Resume`, `SetVolume`, `SetPan`, `SetPitch`, `Playing`.
- **Options** (also used by emitters): `audio.Volume(v)`, `audio.Pan(p)` (−1…1), `audio.Pitch(p)` (resampled, so it also changes speed), `audio.Loop()`, `audio.OnBus("sfx")`, `audio.FadeIn(d)`, `audio.Priority(n)`.
- **Music**: `mixer.PlayMusic(stream, audio.Loop(), audio.Crossfade(2*time.Second))` plays on the `music` bus and crossfades from the current track (equal power). `mixer.StopMusic(audio.FadeOut(d))` stops it.
- **Buses**: `audio.WithBus("music", 0.7)` and `audio.WithBus("sfx", 1)` at creation; `master` always exists. At runtime: `mixer.Bus("music").SetVolume(0.4)`, `.Mute(true)`. A voice's gain is voice × bus × master, smoothed over 10 ms to avoid clicks.
- **Events**: `audio.Finished{Voice}` when a non-looping voice ends, `audio.Looped{Voice}` at each loop, through `world.Emit`, read with a `world.Reader`.

## Positional sound

- `audio.Listener` is a component; add it to the camera entity, or any entity with a transform. The first listener found wins.
- `audio.Emitter` is a component built with `audio.NewEmitter(clip, opts...)`. Its options are the voice options plus `audio.Distance(min, max)`, `audio.Rolloff(r)` and `audio.Autoplay()`. An entity with a transform and an `*audio.Emitter` makes sound where it is.
- The positional system reads `scene.Transform` (3D) or the 2D transform of the entity. Each frame, for every playing emitter:
  - **gain**: OpenAL-style inverse distance, clamped: `min / (min + rolloff·(d − min))`, 1 inside `min`, silent beyond `max`;
  - **pan**: the emitter's position in the listener's local frame, `x / d`, with equal-power panning;
  - in 2D, the same with z = 0.
  - No Doppler or HRTF in the first version.

## API sketch

```go
// examples/32-play-a-sound: press Space to play a sound; Escape quits.
func main() {
	hit, err := audio.Load(assets, "hit.wav") // wav registered by import _ ".../audio/wav"
	if err != nil {
		log.Fatal(err)
	}
	mixer := audio.NewMixer()
	win := platform.NewWindow(platform.Title("ember · play a sound"))
	ember.New().
		AddEntities(win, mixer).
		On(world.Update, "play", func(f world.Frame) {
			if win.Input().Pressed(input.KeySpace) {
				mixer.Play(hit, audio.Pitch(0.9+rand.Float32()*0.2))
			}
		}).
		MustRun()
}
```

```go
// examples/33-music-and-positional: looping music, and a campfire you hear
// louder and on the right side as the camera orbits it.
type Eye struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
	camera.Orbit
	audio.Listener
}

type Campfire struct {
	*world.BasicEntity
	scene.Transform
	render.Model
	*audio.Emitter
}

func main() {
	theme, _ := audio.Open(assets, "theme.ogg")
	crackle, _ := audio.Load(assets, "fire.ogg")
	mixer := audio.NewMixer(audio.WithBus("music", 0.5), audio.WithBus("sfx", 1))
	mixer.PlayMusic(theme, audio.Loop(), audio.FadeIn(time.Second))
	fire := &Campfire{world.NewBasic(), scene.At(3, 0, 0), fireModel(),
		audio.NewEmitter(crackle, audio.Loop(), audio.Autoplay(), audio.Distance(1, 20), audio.OnBus("sfx"))}
	var finished world.Reader[audio.Finished]
	ember.New().
		AddEntities(platform.NewWindow(), mixer, fire, newEye()).
		On(world.Update, "log", func(f world.Frame) {
			for ev := range finished.Read(f.World) {
				log.Println("finished", ev.Voice)
			}
		}).
		MustRun()
}
```

## Tests

- Each decoder: generated files (WAV written in code; tiny Ogg/MP3 fixtures generated with a script and committed only if license-free), truncation at every offset, a fuzz test.
- Mixer with `audio.Recorder`: a one-shot ends and emits `Finished`; two sounds sum; a bus at 0 silences; a crossfade's gains follow the curve; pitch 2 halves the length; voice stealing.
- Positional: gain and pan for emitters left, right, behind, beyond `max`, in 2D and 3D.
- A guard that `ember/audio` and its decoders never import `audiodevice`, `gpu` or `platform/desktop`.

## Open questions

1. Add oto v3.3.2 (one network fetch; cgo only on Linux)? Or a smaller hand-written CoreAudio/WASAPI/ALSA layer through purego? oto is proven and small, so I recommend it.
2. MP3: include it (go-mp3 v0.3.4, needs a fetch) or leave it out? Ogg covers music, WAV covers effects.
3. Should the desktop runner open the device automatically when a `Mixer` is added (my proposal), or should the game pass the output explicitly (`audio.NewMixer(audio.WithOutput(desktop.Audio()))`)? Auto-open matches how windows work today.
4. Do streams decode on their own goroutine with a ring buffer, or inline on the device thread? A goroutine is safer for Vorbis's bursty decode cost.
5. Should a 3D listener's up-vector come from the camera, so a rolled camera pans correctly? I propose yes.
6. Networking: should servers emit sound events for clients to play (a replicated "play at" message)? Not in the first version.

## Effects

Reverb, filters and echo, in pure Go and deterministic, so a Recorder test hears exactly what a player would.

- **Effects:** `audio.NewReverb(RoomSize, Damping, Wet)` (Freeverb: eight comb and four all-pass filters per ear, tuned for 44.1 kHz and scaled to the mixer's rate); `audio.NewLowPass` and `audio.NewHighPass(Cutoff, Resonance)` (RBJ biquads, 12 dB per octave; Resonance is Q, 0.707 by default); `audio.NewEcho(Delay, Feedback, Wet)` (a stereo delay line up to 2 s). Wet is the share of the effect in the output, from 0 (dry) to 1.
- **Options are typed per effect**, so a wrong option fails to compile: `RoomSize` and `Damping` are `ReverbOption`s, `Cutoff` and `Resonance` `FilterOption`s, `Delay` and `Feedback` `EchoOption`s, and `Wet` works on both reverb and echo. Every option has a getter and a setter (0019).
- **Where:** on a bus, `audio.WithBus("sfx", 1, audio.Effects(reverb))` or `mixer.Bus("sfx").SetEffects(...)`, and on a voice or an emitter, `mixer.Play(clip, audio.Effects(lowPass))`. They run in the order given: a voice's on that voice alone, then its bus's on the bus's mix, then master's on everything. A bus's reverb tail rings on after its voices end.
- **No clicks:** setters only set a target. Each effect moves its parameters there over 10 ms, in steps of 32 frames, like voice gains, so a game can change or tween them every frame (`ease.NewTween(1, func(t float32) { reverb.SetRoomSize(...) })`).
- **One owner:** an effect keeps its own state (delay lines, filter memory), so one effect value belongs to one bus or one voice. Give each voice its own.
- **Closed set for now:** `Effect` is an interface the package implements. Opening it to game effects means exposing the block size and rate contract, which can wait for a request.
- **Cost:** the mixer works in blocks; a voice with effects mixes into a scratch buffer first. Budget: 32 voices with a filter each plus a bus reverb mix a 60 Hz frame (800 frames at 48 kHz) in well under a millisecond, which `BenchmarkMixWithEffects` checks.

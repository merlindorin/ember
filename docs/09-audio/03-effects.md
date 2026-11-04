# Effects

Sound changes with the place it is heard in. Here a corridor opens into a great hall: the hero's footsteps and claps ring longer as he walks in, and drums play muffled behind a door until he opens it. The lines at the top show the reverb and the filter, since a GIF can't play them. Run it to hear it.

![The hero walks out of the corridor into the hall, the reverb growing, then opens the door and the drums' filter opens](effects.gif "ember · audio effects")

## The effects

ember has four effects, each made with its own options:

- `audio.NewReverb(RoomSize, Damping, Wet)`: the sound of a room, from a small one to a large one.
- `audio.NewLowPass(Cutoff, Resonance)`: lets the low sounds through and muffles the high ones, as through a wall or under water. `audio.NewHighPass` does the opposite, as through a radio.
- `audio.NewEcho(Delay, Feedback, Wet)`: the sound repeated, fading.

Every option has a getter and a setter of the same name, `Cutoff()` and `SetCutoff`, so an effect can change while it plays. A change glides over 10 ms by itself, so setting an effect every frame never clicks.

## On a bus, or on one sound

`audio.Effects(…)` puts effects on a bus, with `WithBus`, or on one sound, with `Play`, `PlayMusic` or `NewEmitter`. A sound goes through its own effects first, then its bus's, then the master bus's. A bus's reverb rings on after its sounds end; a sound's own effects end with it. An effect belongs to one bus or one sound. `Bus.SetEffects` and `Voice.SetEffects` change them while the game runs.

Here the sound effects bus has a reverb, so every footstep and clap rings in the room the hero is in.

[main.go](../../examples/54-audio-effects/main.go)

## A room that changes

`Acoustics` blends the reverb from the corridor's small room to the hall's large one over the archway, setting its size and wet every frame from where the hero stands.

[acoustics.go](../../examples/54-audio-effects/acoustics.go)

## Behind the door

The drums are an emitter with its own low-pass, at 350 Hz while the door is shut. E near the door opens it, and an `ease` tween sweeps the cutoff up to 16 kHz over half a second, so the drums come through bright. Shutting the door sweeps it back down.

[door.go](../../examples/54-audio-effects/door.go)

## The rest

The hero walks with WASD, a footstep every third of a second, and claps on Space; both play on the sound effects bus. The dungeon is two polygons, and the HUD prints the room, the reverb and the cutoff.

[hero.go](../../examples/54-audio-effects/hero.go)

[dungeon.go](../../examples/54-audio-effects/dungeon.go)

[status.go](../../examples/54-audio-effects/status.go)

The footsteps, the clap and the drums are synthesised by the example's own generator, free to use (CC0).

## Try it

[54-audio-effects](../../examples/54-audio-effects)

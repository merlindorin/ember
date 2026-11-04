# Music and positional sound

A day forest ambience loops over the meadow, and M crossfades it to a night one with owl calls, the meadow darkening with it, and back. A campfire crackles on the left and a river runs on the right: walking to one makes it louder, and each is heard from the side it is on, which changes as the hero turns. The lines at the top show what the hero hears of each, since a GIF can't. Run it to hear it.

![The hero turns a full circle between the campfire and the river, the pans swapping sides, then walks to the river](music-and-positional.gif "ember · music and positional sound")

## Music

`audio.Open` opens a stream, here an Ogg Vorbis recording, decoded as it plays rather than all at once, as suits anything long. The mixer is made with buses, `WithBus("music", 0.5)` and `WithBus("sfx", 1)`: each voice plays on one, and a bus's volume and mute apply to all of them at once, which is how a settings menu turns music down. `PlayMusic` plays a stream on the music bus, one track at a time, here looping and fading in over a second. A track needn't be music: here the day and night ambiences are, since one plays at a time in the background.

[main.go](../../examples/53-music-and-positional/main.go)

## The crossfade

Playing another track with `Crossfade` fades the one playing out while the new one fades in, over the same two seconds as the meadow's colour. Each track plays at its own `Volume`, to balance them.

[music.go](../../examples/53-music-and-positional/music.go)

## Sounds in the meadow

The hero carries an `audio.Listener`: the meadow is heard from where he stands. The campfire and the river each carry an `audio.Emitter`, a sound that plays where its entity is. Both are streams, opened with `audio.Open` like the ambiences, since long recordings are better read as they play; an emitter takes a clip from `audio.Load` just the same. `Autoplay` starts it with the game, `Loop` keeps it going, and `OnBus` plays it on the sound effects bus. `Distance(min, max)` is how near it is heard in full and how far it fades to silence, and `Rolloff` how quickly it fades in between. Tune them per source: the fire, `Distance(30, 500)` and `Rolloff(1)`, carries less far than the river, `Distance(80, 700)` and `Rolloff(0.7)`, so each one dominates near itself. Its pan comes from where it is beside the listener, and `Heard()` says how loud and from which side the listener hears it, which the HUD prints.

[scene.go](../../examples/53-music-and-positional/scene.go)

## Turning the listener

A listener hears from where its entity faces, so turning the hero turns his ears. In 2D, a listener at rotation 0 faces up the screen, and turning it clockwise turns its ears with it: after a half turn, the fire on the left is heard on the right. Q and E turn the hero, and walking turns him to face his way; a yellow arrow shows where he faces, and the HUD's pans change as he turns. `Face` turns the hero but keeps his sprite upright, mirrored to the side he faces, so only the ears and the arrow turn. In 3D, a listener faces down its transform's -Z, so it goes on the camera. The hero walks as on the [Keyboard](../08-input/01-keyboard.md) page, kept inside the window.

[walk.go](../../examples/53-music-and-positional/walk.go)

## Credits

The sounds are recordings from Freesound, all under CC0 1.0:

- the river: ["Little river" by AudioPapkin](https://freesound.org/people/audiopapkin/sounds/815124/)
- the campfire: ["Campfire 01" by HECKFRICKER](https://freesound.org/people/heckfricker/sounds/729395/)
- the day: ["ForestDay_BW.66" by sonidosreales245](https://freesound.org/people/sonidosreales245/sounds/517042/)
- the night: ["Forest owl calls in the night city atmosphere" by Garuda1982](https://freesound.org/people/garuda1982/sounds/721046/)

## Try it

[53-music-and-positional](../../examples/53-music-and-positional)

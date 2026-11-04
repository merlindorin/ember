# Play a sound

Space plays a short pling, a little higher or lower each time, and a click plays it from the side of the window it lands on. A ring shows where each one played, and the line at the top counts the sounds playing and finished. Run it to hear it.

![Rings where the sound plays: at a click on the left, in the middle, at a click on the right](play-a-sound.gif "ember · play a sound")

## The mixer

`audio.Load` reads a clip, here a WAV file embedded with the example, which the `audio/wav` import can decode. The mixer is added to the World with the window, as an entity like it.

[main.go](../../examples/52-play-a-sound/main.go)

## Playing it

`mixer.Play` starts a voice of a clip, and options shape it: `Pitch` plays it faster and higher, or slower and lower, and `Pan` puts it between the left ear, -1, and the right, 1. The same clip can play as many voices at once as needed; `Voices()` counts those playing. When a voice ends, the World gets an `audio.VoiceFinished` event, read like any other. `Play` also returns the voice, to `Stop` it, with a `FadeOut` if wanted, `Pause` and `Resume` it, or change its volume, pan and pitch while it plays.

[play.go](../../examples/52-play-a-sound/play.go)

## Try it

[52-play-a-sound](../../examples/52-play-a-sound)

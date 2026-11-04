# Audio

Sound is played by a mixer, `audio.NewMixer()`, an entity the game adds to the World next to its window. Short sounds are clips, decoded whole into memory by `audio.Load`; music and long ambiences are streams, `audio.Open`, decoded as they play. Formats come as imports, `_ "github.com/merlindorin/ember/audio/wav"` and `.../audio/vorbis`, so a game only builds in the decoders it uses.

A page can't play sound: its GIF shows what the example draws while it plays, so run the example to hear it.

## Silent output

The mixer needs no sound device. `ember.New` installs audio, and the device opens only when a game runs in a window; a server, a test or a hidden capture mixes into a silent output at the World's pace instead, so voices still play, end and send their events. In a test, `audio.NewRecorder(mixer)` keeps what was mixed: after `RunFrames`, `Samples()` holds it, and `Peak` and `PeakLR` say how loud it was, and on which side. Streams decode at once there, so a test hears the same thing every run.

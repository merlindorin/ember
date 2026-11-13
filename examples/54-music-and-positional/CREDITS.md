# Credits

Every sound here is a CC0 1.0 recording from Freesound, cut into a seamless loop and normalised. Only the cut, compressed loops are in the repository, never the source recordings.

| File | Recording | Author | Source | Start (s) | Length (s) | Fade (s) | Loudness (LUFS) |
|---|---|---|---|---|---|---|---|
| `river.ogg` | Little river | AudioPapkin | https://freesound.org/people/audiopapkin/sounds/815124/ | 0.5 | 43.5 | 1.5 | -23 |
| `crackle.ogg` | Campfire 01 | HECKFRICKER | https://freesound.org/people/heckfricker/sounds/729395/ | 20 | 34 | 1.5 | -23 |
| `day.ogg` | ForestDay_BW.66 | sonidosreales245 | https://freesound.org/people/sonidosreales245/sounds/517042/ | 40 | 42 | 2 | -26 |
| `night.ogg` | Forest owl calls in the night city atmosphere | Garuda1982 | https://freesound.org/people/garuda1982/sounds/721046/ | 0.5 | 35 | 2 | -26 |

Each loop takes LENGTH seconds from START, at 44.1 kHz stereo (day's 6-channel 96 kHz source is downmixed), normalises it to LOUDNESS with at most -2 dBTP of true peak, and joins its end to its start with an equal-power crossfade of FADE seconds, encoded as Vorbis quality 5. With END = START + LENGTH + FADE, and TAIL = LENGTH + FADE, a first pass measures the loudness:

```sh
ffmpeg -i source.wav -af "atrim=start=START:end=END,asetpts=PTS-STARTPTS,aresample=44100,aformat=channel_layouts=stereo,loudnorm=I=LOUDNESS:TP=-2:LRA=11:print_format=json" -f null -
```

and the second applies those measurements (`input_i`, `input_tp`, `input_lra`, `input_thresh`, `target_offset`) and cuts the loop:

```sh
ffmpeg -i source.wav -filter_complex "[0:a]atrim=start=START:end=END,asetpts=PTS-STARTPTS,aresample=44100,aformat=channel_layouts=stereo,loudnorm=I=LOUDNESS:TP=-2:LRA=11:measured_I=…:measured_TP=…:measured_LRA=…:measured_thresh=…:offset=…:linear=true,aresample=44100,asplit=3[a][b][c];[a]atrim=start=FADE:end=LENGTH,asetpts=PTS-STARTPTS[main];[b]atrim=start=LENGTH:end=TAIL,asetpts=PTS-STARTPTS,afade=t=out:st=0:d=FADE:curve=qsin[tail];[c]atrim=start=0:end=FADE,asetpts=PTS-STARTPTS,afade=t=in:st=0:d=FADE:curve=qsin[head];[tail][head]amix=inputs=2:normalize=0[join];[main][join]concat=n=2:v=0:a=1[out]" -map "[out]" -ac 2 -c:a libvorbis -q:a 5 out.ogg
```

loudnorm falls back to its dynamic mode where a linear gain would break the peak limit, as on the campfire's sharp pops, so the results land within 2 LU of their targets.

`walker.png` is the engine's walker sprite.

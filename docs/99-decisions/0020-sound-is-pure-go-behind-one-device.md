# 0020. Sound is pure Go behind one device

- Status: Accepted
- Date: 2026-10-29
- Area: core
- Owner: user

## Context

Games need effects, music and positional sound, while servers and tests run without a sound card and must stay pure Go (0003, 0004, 0009).

## Decision

- `audio` mixes clips, streams, buses and emitters in pure Go; `audio/wav` is our own decoder and `audio/vorbis` wraps jfreymuth/oggvorbis. No MP3.
- The only device code is `internal/audiodevice`, on ebitengine/oto v3: no cgo on macOS and Windows, ALSA with cgo on Linux. The desktop runner starts it for the first `audio.Mixer`; hidden windows play no sound.
- Without a device, a mixer mixes into a null output at the World's pace, so voices end and events fire on servers too; `audio.Recorder` keeps those samples for tests, and streams then decode synchronously.
- Emitters are placed by their entity's 3D transform or 2D placement and heard from the first `audio.Listener`, with clamped inverse-distance gain and equal-power pan; no Doppler or HRTF.

## Alternatives

- Hand-written CoreAudio/WASAPI/ALSA bindings through purego: fewer dependencies, but three platforms to maintain where oto is small and proven.
- The game passing an output to the mixer: explicit, but unlike windows, which the runner opens by itself.

## Consequences

Sound costs three dependencies (oto, purego, oggvorbis) and nothing on servers. Tests assert on what was mixed, frame by frame, with no device and no real time.

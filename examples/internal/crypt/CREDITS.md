# Credits

- `art/*.png`: 61 frames from [16x16 DungeonTileset II](https://0x72.itch.io/dungeontileset-ii) v1.7 by 0x72, under CC0 ("You can use this tileset for whatever you like"). Unchanged.
- `sound/music.ogg`: "8-bit - Perilous Dungeon" by HydroGene, https://opengameart.org/content/8-bit-perilous-dungeon, CC0 1.0. The whole 112.5 s track, which loops seamlessly, with its first 955 samples (MP3 encoder delay) trimmed, normalised to -23 LUFS and encoded mono at 22.05 kHz:

  ```sh
  ffmpeg -i 8-bit_perilous_dungeon.mp3 -af "atrim=start_sample=955,asetpts=PTS-STARTPTS,aformat=channel_layouts=mono,loudnorm=I=-23:TP=-2:LRA=11:print_format=json" -f null -
  ffmpeg -i 8-bit_perilous_dungeon.mp3 -af "atrim=start_sample=955,asetpts=PTS-STARTPTS,aformat=channel_layouts=mono,loudnorm=I=-23:TP=-2:LRA=11:measured_I=-11.40:measured_TP=2.46:measured_LRA=2.70:measured_thresh=-21.40:offset=0.11:linear=true,aresample=22050" -c:a libvorbis -q:a 0 music.ogg
  ```

- `sound/shoot.ogg`, `hit.ogg`, `kill.ogg`, `powerup.ogg`, `hurt.ogg`, `gameover.ogg`: synthesised by `sounds_gen.go`, CC0 1.0.

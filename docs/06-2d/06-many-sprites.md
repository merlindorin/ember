# Many sprites

A bunnymark: hundreds of bunnies falling and bouncing around the window, and a hundred more every frame while Space or the mouse button is held. The counter shows how many there are and the frame rate, so you can see how far your machine goes.

![Bunnies bouncing, and more pouring in from the corner](bunnies.gif "ember · many sprites")

## The program

The bunnies are drawn on one layer and the counter on a layer above it, so the counter stays readable. All the bunnies share one texture.

[main.go](../../examples/24-many-sprites/main.go)

## Bunnies

`Bunnies` adds them: `Add` makes each sprite a child of the field layer, and since the layer is already in the World, the sprite joins it too. Every frame, each bunny falls with gravity, moves by its speed with `Translate`, and bounces off the edges through `Position` and `SetPosition`. `f.World.Stats().FPS` is the frame rate.

[bunnies.go](../../examples/24-many-sprites/bunnies.go)

## Counting draws

F3 in any ember window opens the profiler: a frame graph, the entity count, the slowest systems, and a line for 2D, "2D draws N  binds N  verts Nk  passes N". Its "work" line is the frame's own time, its tick and render without the wait for the display, and the frame rate that would allow, "work 1.60ms (~625 fps possible)": under vsync the frame rate is the display's, so that's where the room to spare shows. `twod.ShowProfiler(w, on)` shows or hides it from code, the same switch F3 toggles, so a game's own debug menu and F3 agree, and `twod.ProfilerShown(w)` reads it. Draws are the GPU calls of the last frame, binds the texture changes between them, verts the corners drawn, and passes the pictures drawn first offscreen: cameras with a `Resolution` or `Into` a texture, masks and blended layers. `Stats().Work()` is that work time. The same counts are `Stats().Draws2D`, `Binds2D`, `Vertices2D` and `Passes2D`. Below the slowest systems, a "recorded" list shows what no system covers: collecting and drawing the 2D, and the GPU's own time, for the window's 2D, the scene, and each offscreen pass, named after its layer or camera, so `twod.Name("light")` on a lighting layer shows "gpu pass light". A "gpu frame" row gives the GPU's time for the whole frame; on Apple GPUs the passes overlap, so their rows can add up to more than it. The GPU is timed only while the profiler shows, and its times are read back four frames late, so measuring them never stalls the frame; a "swap" line shows the time spent handing the frame to the screen. For a measure without the overlay, `EMBER_STATS=1 go run …` prints, on quitting, the frame's work and the frame rate it would allow, the frame, tick, render and swap times, the wait for the display as "acquire" (left out of its reports of slow frames) and each system's and GPU pass's cost, averaged after a 60-frame warm-up; `EMBER_STATS=cpu` leaves the GPU untimed. It also names the backend and the graphics card it drew with. A game records its own spans with `Stats().Record(name, d)`, smoothed like the rest, and `Recorded()` lists them; `Stats().Count(name)` adds one to a running counter, which `Counted()` lists and the profiler shows under "counted". One draw call takes sprites of up to eight textures, even interleaved, as in a layer sorted by y where a hero stands between trees between enemies of several sheets. A new call starts only for a ninth texture, or a change of clip, blend or mask, and the draw order stays as it was; so a typical scene draws in a handful of calls, and beyond eight sheets mixed together, more sheets mean more draws ([decision 0031](../99-decisions/0031-a-batch-draws-several-textures.md)).

What no camera shows isn't drawn. A sprite, nine-patch, line or polygon whose bounds, with a 16-point margin, lie outside every camera that draws its layer, the main one and any drawing `Into` a texture such as a minimap, is skipped, and a layer sorted by y sorts only what is in view. So a world can hold far more sprites than the screen shows, and the ones off screen cost nothing to draw. Fixed layers and the UI always draw, and so does what reports no bounds: a custom `Draw`, text and particles. Tile maps cut their own cells to the view.

## Try it

[24-many-sprites](../../examples/24-many-sprites)

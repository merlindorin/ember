# Hello cube

The smallest program that opens a window and draws: a spinning cube whose corners are coloured by their position. [The World](../02-world/README.md) explains the entities and systems it uses.

![The spinning RGB cube](hello-cube.png "ember · hello cube")

## The program

The window is an entity made with `platform.NewWindow`, its title, size and background. `ember.New` returns a World; the window, the cube and the camera are entities in it, and a function added with `On` turns the cube a little every frame.

[main.go](../../examples/01-hello-cube/main.go)

## Try it

[01-hello-cube](../../examples/01-hello-cube)

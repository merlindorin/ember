# Lines and shapes

Lines and filled shapes are 2D entities like sprites, made from a list of points. Here three zigzags show the ways a thick line can turn and end, with a star outline and a filled arrow below them.

![Three zigzags, a star outline and a red arrow](lines.png "ember · lines and shapes")

## The program

`twod.NewLine` draws through its points with a `Width`. `Joints` says how it turns at each point and `Caps` how its ends look; from left to right, the zigzags are `Miter` with `Flat` caps, `Bevel` with `Square` caps, and `RoundJoint` with `RoundCap` ends. `Closed` joins a line's last point back to its first, for the star. `twod.NewPolygon` fills a shape, concave ones such as the arrow included. Points are relative to the entity's `At`, and `Color` paints it.

[main.go](../../examples/21-lines-and-shapes/main.go)

## Try it

[21-lines-and-shapes](../../examples/21-lines-and-shapes)

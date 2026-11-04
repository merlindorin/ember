# Parents and children

An entity can own other entities. Here two sparks belong to the campfire: when it goes out at 1s they leave with it, though they would burn for 5s on their own.

## Adding children

`AppendChild` makes each spark a child of the campfire. Adding the campfire to the World adds both sparks, which the count at `Startup` shows.

[main.go](../../examples/05-children/main.go)

## Reaching them

`Children()` lists a parent's children, and `Descendents()` their children too, all the way down. Each is the very entity that was appended, so a type assertion finds a game's own kind among them; a child appended as a bare `BasicEntity` comes back as its `*BasicEntity`. The slice is new each call, so changing it changes nothing:

```go
for _, c := range hero.Children() {
	if fx, ok := c.(*Effect); ok {
		fx.Left -= dt
	}
}
```

`RemoveChild` only detaches a child: it stays in the World, on its own. To drop it for good, remove it from the World too. `world.Of(e)` is the World an entity is in, or nil:

```go
campfire.RemoveChild(spark.BasicEntity)
if w := world.Of(spark); w != nil {
	w.RemoveEntity(spark)
}
```

## As before

Nothing else knows about children: `Cooling` removes the campfire, and the World removes its sparks.

[cooling.go](../../examples/05-children/cooling.go)

[components.go](../../examples/05-children/components.go)

[flame.go](../../examples/05-children/flame.go)

## Try it

[05-children](../../examples/05-children)

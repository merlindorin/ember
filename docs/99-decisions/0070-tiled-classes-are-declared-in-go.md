# 0070. Tiled classes are declared in Go

- Status: Accepted
- Date: 2026-11-13
- Area: assets
- Owner: user

## Context

Tiled saves only the class members that were changed; the defaults live in the `.tiled-project` file. A loader that
doesn't know a class reads nothing for a member left at its default, so the class has to be known at load. Checked
with Tiled 1.12.2: a member left alone is absent from the file, a member set to a value equal to its default stays,
and an export with types and properties resolved writes every member.

## Decision

- A game declares its classes in Go: `twod.Classes(twod.Class{Name, On, Shapes, Properties, Doc})`, each property a
  `twod.Property` (once `TileProperty`), with `KindClass`, `KindFile` and `KindObject` beside the plain kinds.
- At load, a map, layer or object of a declared class has every property of the class: the file's value, else the
  default. A tile reads its class's defaults when asked (`Tileset.Value`); the engine's and the game's tile properties
  stay on every tile, the class adds to them. The three ways Tiled saves a class load the same.
- Once a class is declared for a use, an undeclared class there, a property the class lacks, a wrong kind or range, a
  `Required` property left empty and an object of a shape the class doesn't take are errors. Nothing declared, nothing
  checked.
- `twod.TiledProject(opts…)` writes the classes for Tiled, a tile class with `ember.Tile`'s members first. A game
  tests that its project file is what its declarations write.

## Alternatives

- Read the `.tiled-project` at load: a designer adds a class without Go, but nothing checks the game reads it, the
  file must ship with the game, and it breaks decision 0043's Go-first tile properties.
- Fill nothing and leave defaults to the game: every game repeats Tiled's defaults by hand, and they drift.

## Consequences

Go is the source and Tiled the editor: a class added in Tiled's type editor is overwritten when the project file is
written, and refused at load. A changed default changes every map that left the value alone, with no diff in the
levels.

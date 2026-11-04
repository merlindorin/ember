package main

// The tiles of tiles.png, numbered from 1, left to right then top to
// bottom.
const (
	grass = iota + 1
	wall
	water
	ripple
	tree
	pillar
	sand
	floor
)

// legend gives the tile each character of the level stands for.
func legend() map[rune]int {
	return map[rune]int{'.': grass, '#': wall, '~': water, '=': sand, '_': floor}
}

const level = `
....................
..~~~~~.............
.~~~~~~~.....######.
.~~~~~~~=====#____#.
..~~~~~.=....#____#.
........=....#____#.
........=....##_###.
........=......=....
........=========...
....................
....................
....................`

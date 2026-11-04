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

func legend() map[rune]int {
	return map[rune]int{'.': grass, '#': wall, '_': floor, '=': sand}
}

// room is a walled garden and a stone room side by side, with grass around
// them; the wall between them has a door at doorX, doorY, shut until the
// hero steps on the sand plate at plateX, plateY.
const room = `
....................
.##################.
.#.....#__________#.
.#.....#__________#.
.#.....#__________#.
.#.....#__________#.
.#.....#__________#.
.#.....#__________#.
.#...=.#__________#.
.#.....#__________#.
.##################.
....................`

const (
	doorX, doorY   = 7, 8
	plateX, plateY = 5, 8
)

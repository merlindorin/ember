package main

import "github.com/merlindorin/ember/input"

// Act is what the player means to do, whatever button says it.
type Act int

const (
	Left Act = iota
	Right
	Up
	Down
	PlantTree
)

// String names an action in saved bindings.
func (a Act) String() string {
	return [...]string{"left", "right", "up", "down", "plant"}[a]
}

// newControls binds each action to its default buttons: a key on either
// hand, a mouse button, or a pad's stick, cross and south button.
func newControls() *input.Actions[Act] {
	return input.NewActions(
		input.Bind(Left, input.KeyLeft, input.KeyA, input.GamepadLeftStickLeft, input.GamepadLeft),
		input.Bind(Right, input.KeyRight, input.KeyD, input.GamepadLeftStickRight, input.GamepadRight),
		input.Bind(Up, input.KeyUp, input.KeyW, input.GamepadLeftStickUp, input.GamepadUp),
		input.Bind(Down, input.KeyDown, input.KeyS, input.GamepadLeftStickDown, input.GamepadDown),
		input.Bind(PlantTree, input.MouseLeft, input.KeyEnter, input.GamepadSouth),
	)
}

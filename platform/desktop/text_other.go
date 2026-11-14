//go:build !darwin

package desktop

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/input"
)

// listenText leaves typed text to GLFW's character callback.
func listenText(*glfw.Window, *input.State) {}

func forgetText(*glfw.Window) {}

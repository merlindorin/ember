//go:build !darwin

package desktop

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/input"
)

// listenTouches waits for a backend: touches come from macOS only for now,
// and elsewhere a window simply reports none.
func listenTouches(*glfw.Window, *input.State) {}

func forgetTouches(*glfw.Window) {}

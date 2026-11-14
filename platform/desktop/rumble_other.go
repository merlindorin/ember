//go:build !darwin

package desktop

import "github.com/lantervale/ember/internal/inputfeed"

// playRumble waits for a backend: pads rumble on macOS only for now, and
// elsewhere they stay still.
func playRumble(int, int, inputfeed.Rumble) {}

func stopRumbles() {}

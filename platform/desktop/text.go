//go:build darwin

package desktop

import "github.com/lantervale/ember/input"

// addText types text into in: every character but controls and the
// private range macOS gives function keys, emoji and their joiners kept.
func addText(in *input.State, text string) {
	for _, r := range text {
		if r < 32 || r == 127 || r >= 0x80 && r < 0xa0 || r >= 0xf700 && r <= 0xf7ff {
			continue
		}
		in.AddText(r)
	}
}

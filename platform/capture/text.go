package capture

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lantervale/ember/input"
)

// typingEvery is how many frames apart typed characters land, about
// fifteen a second at sixty frames.
const typingEvery = 4

// Text types strings into the window, a character at a time, as a player
// would.
type Text []Typing

// Typing is Text typed from frame From on.
type Typing struct {
	Text string
	From int
}

// ParseText reads a script such as "Heron@60;ok@120": each text is typed
// from its frame on, one character every four frames.
func ParseText(script string) (Text, error) {
	var t Text
	for part := range strings.SplitSeq(script, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		i := strings.LastIndex(part, "@")
		if i <= 0 {
			return nil, fmt.Errorf("capture: %q is not text@frame", part)
		}
		from, err := strconv.Atoi(strings.TrimSpace(part[i+1:]))
		if err != nil || from < 0 {
			return nil, fmt.Errorf("capture: %q is not text@frame", part)
		}
		t = append(t, Typing{part[:i], from})
	}
	return t, nil
}

// Apply types the characters due at frame through in, before the frame
// begins, so the frame sees them.
func (t Text) Apply(in *input.State, frame int) {
	for _, ty := range t {
		n := frame - ty.From
		if n < 0 || n%typingEvery != 0 {
			continue
		}
		if r := []rune(ty.Text); n/typingEvery < len(r) {
			in.AddText(r[n/typingEvery])
		}
	}
}

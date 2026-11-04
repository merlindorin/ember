// Package capture records what a window shows for documentation: an
// animated GIF of several frames, and the keys held while it records. It is
// pure Go; the desktop runner reads the framebuffer and feeds it here.
package capture

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/input"
)

// Keys holds keys down during frame ranges, as a player would.
type Keys []Hold

// Hold is a key held from frame From up to, but not including, frame To.
type Hold struct {
	Key      input.Key
	From, To int
}

// ParseKeys reads a script such as "right:0-40; left:40-80": each key, by
// its name in any case, is held from the first frame up to the second.
// Entries are separated by ";", like every capture script; "," still works.
func ParseKeys(script string) (Keys, error) {
	var keys Keys
	split := func(r rune) bool { return r == ';' || r == ',' }
	for _, part := range strings.FieldsFunc(script, split) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, frames, ok := strings.Cut(part, ":")
		from, to, ok2 := strings.Cut(frames, "-")
		if !ok || !ok2 {
			return nil, fmt.Errorf("capture: key entry %q is not key:from-to (entries are separated by ;)", part)
		}
		k, ok := keyNamed(name)
		if !ok {
			return nil, fmt.Errorf("capture: key entry %q: no key named %q", part, name)
		}
		f, err1 := strconv.Atoi(strings.TrimSpace(from))
		t, err2 := strconv.Atoi(strings.TrimSpace(to))
		if err1 != nil || err2 != nil || f < 0 || t <= f {
			return nil, fmt.Errorf(
				"capture: key entry %q needs whole frames from < to (entries are separated by ;)",
				part,
			)
		}
		keys = append(keys, Hold{k, f, t})
	}
	return keys, nil
}

// Apply sets every scripted key down or up for frame, through in, as the
// window's own key events would. A key held more than once is down while
// any of its holds lasts.
func (ks Keys) Apply(in *input.State, frame int) {
	down := make(map[input.Key]bool)
	for _, h := range ks {
		down[h.Key] = down[h.Key] || frame >= h.From && frame < h.To
	}
	for k, d := range down {
		in.SetKey(k, d)
	}
}

func keyNamed(name string) (input.Key, bool) {
	for k := input.Key(0); k <= input.KeyMenu; k++ {
		if strings.EqualFold(k.String(), strings.TrimSpace(name)) {
			return k, true
		}
	}
	return 0, false
}

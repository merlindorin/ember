package node

import (
	"fmt"
	"slices"
	"strings"
)

// Words names the values of an enum in scene files, by value: Words[T]{"a",
// "b"} stores 0 as "a" and 1 as "b". An enum's MarshalText and
// UnmarshalText call Text and Parse.
type Words[T ~int | ~uint8] []string

func (w Words[T]) Text(v T) ([]byte, error) {
	if int(v) < 0 || int(v) >= len(w) {
		return nil, fmt.Errorf("node: %d is not one of %s", v, strings.Join(w, ", "))
	}
	return []byte(w[v]), nil
}

func (w Words[T]) Parse(v *T, text []byte) error {
	i := slices.Index(w, string(text))
	if i < 0 {
		return fmt.Errorf("node: %q is not one of %s", text, strings.Join(w, ", "))
	}
	*v = T(i)
	return nil
}

// FlagWords names a bit set: bit i of a value is Bits[i], and a value
// is stored as its set bits joined by |, or as None when it has none.
type FlagWords[T ~int | ~uint8] struct {
	None string
	Bits []string
}

func (w FlagWords[T]) Text(v T) ([]byte, error) {
	if v == 0 {
		return []byte(w.None), nil
	}
	var out []string
	for i, name := range w.Bits {
		if v&(1<<i) != 0 {
			out = append(out, name)
			v &^= 1 << i
		}
	}
	if v != 0 {
		return nil, fmt.Errorf("node: bits %b have no name", v)
	}
	return []byte(strings.Join(out, "|")), nil
}

func (w FlagWords[T]) Parse(v *T, text []byte) error {
	*v = 0
	if string(text) == w.None {
		return nil
	}
	for _, name := range strings.Split(string(text), "|") {
		i := slices.Index(w.Bits, name)
		if i < 0 {
			return fmt.Errorf("node: %q is not one of %s, %s", name, w.None, strings.Join(w.Bits, ", "))
		}
		*v |= 1 << i
	}
	return nil
}

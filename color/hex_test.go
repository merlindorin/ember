package color_test

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/merlindorin/ember/color"
)

// oldParseHex is ParseHex as it was before it stopped allocating, kept as
// the reference the new one must match.
func oldParseHex(s string) (color.Color, error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) == 6 {
		h += "ff"
	}
	if len(h) != 8 {
		return color.Color{}, fmt.Errorf("color: invalid hex %q", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return color.Color{}, fmt.Errorf("color: invalid hex %q: %w", s, err)
	}
	c := color.RGB8(uint8(v>>24), uint8(v>>16), uint8(v>>8))
	c.A = float32(uint8(v)) / 255
	return c, nil
}

func TestParseHexMatchesTheOldParserOnEveryForm(t *testing.T) {
	inputs := []string{
		"#fff", "fff", "#FFF", "#a1B", "#000", "#d04030", "d04030", "#D04030", "#AbCdEf",
		"#d0403080", "d0403080", "#FFFFFFFF", "#00000000", "#12345678",
		"", "#", "##fff", "#ff", "#ffff", "#fffff", "#fffffff", "#fffffffff", "#ggg", "#d0403g",
		"#d04030 ", " #d04030", "#+12345", "#-12345", "#0x1234", "#12_456", "#d0403080ff", "#é12",
	}
	random := rand.New(rand.NewPCG(3, 4))
	alphabet := "0123456789abcdefABCDEFgxz#_+- "
	for range 2000 {
		b := make([]byte, random.IntN(11))
		for i := range b {
			b[i] = alphabet[random.IntN(len(alphabet))]
		}
		inputs = append(inputs, string(b))
	}
	for _, s := range inputs {
		want, wantErr := oldParseHex(s)
		got, err := color.ParseHex(s)
		if (err != nil) != (wantErr != nil) || got != want {
			t.Fatalf("%q: %v, %v; the old parser %v, %v", s, got, err, want, wantErr)
		}
	}
}

func TestHexAllocatesNothing(t *testing.T) {
	for _, s := range []string{"#fff", "#d04030", "#d0403080"} {
		if n := testing.AllocsPerRun(100, func() { _ = color.Hex(s) }); n != 0 {
			t.Fatalf("%s: %v allocations", s, n)
		}
	}
}

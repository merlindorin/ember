package desktop

import (
	"slices"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
)

// The pad families matched between GLFW's joysticks and the system's
// haptics, which name a pad by its kind rather than its USB IDs.
const (
	familyOther = iota
	familyDualShock4
	familyDualSense
	familyXbox
)

//nolint:gochecknoglobals // lookup tables
var (
	dualShock4s = []string{"c405", "cc09", "a00b"}
	dualSenses  = []string{"e60c", "f20d"}
)

// padFamily is the family of the pad with an SDL-style GUID, vendor and
// product little endian as in touchpadPad.
func padFamily(guid string) int {
	if len(guid) < 20 {
		return familyOther
	}
	vendor, product := guid[8:12], guid[16:20]
	switch {
	case vendor == sony && slices.Contains(dualShock4s, product):
		return familyDualShock4
	case vendor == sony && slices.Contains(dualSenses, product):
		return familyDualSense
	case vendor == "5e04":
		return familyXbox
	}
	return familyOther
}

// rumbleTarget is the family of pad id, among the GUIDs of the connected
// pads by slot ("" for none), and its rank among the pads of that family:
// the system lists them in the order they connected, as GLFW fills slots.
func rumbleTarget(guids []string, id int) (int, int, bool) {
	if id < 0 || id >= len(guids) || guids[id] == "" {
		return 0, 0, false
	}
	family, nth := padFamily(guids[id]), 0
	for _, g := range guids[:id] {
		if g != "" && padFamily(g) == family {
			nth++
		}
	}
	return family, nth, true
}

// rumbleGamepads plays the rumbles the game asked of in's pads.
func rumbleGamepads(in *input.State) {
	rumbles := inputfeed.Of(in).Rumbles()
	if len(rumbles) == 0 {
		return
	}
	guids := make([]string, input.MaxGamepads)
	for i := range guids {
		if joy := glfw.Joystick1 + glfw.Joystick(i); joy.IsGamepad() {
			guids[i] = joy.GetGUID()
		}
	}
	for _, r := range rumbles {
		if family, nth, ok := rumbleTarget(guids, r.Pad); ok {
			playRumble(family, nth, r)
		}
	}
}

package desktop //nolint:testpackage // tests the unexported rumble targeting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPadsAreGroupedByFamilyForHaptics(t *testing.T) {
	assert.Equal(t, familyDualShock4, padFamily("030000004c050000cc09000000010000"))
	assert.Equal(t, familyDualShock4, padFamily("030000004c050000c405000000010000"))
	assert.Equal(t, familyDualSense, padFamily("050000004c050000e60c000000010000"))
	assert.Equal(t, familyXbox, padFamily("030000005e040000e002000000000000"))
	assert.Equal(t, familyOther, padFamily("030000004c0500006802000000010000"), "a PS3 pad is no DualShock 4")
	assert.Equal(t, familyOther, padFamily("short"))
}

func TestAPadRumblesAsTheNthOfItsFamily(t *testing.T) {
	ds4 := "030000004c050000cc09000000010000"
	dualSense := "050000004c050000e60c000000010000"
	xbox := "030000005e040000e002000000000000"
	guids := []string{ds4, xbox, "", ds4, dualSense}

	family, nth, ok := rumbleTarget(guids, 3)
	assert.True(t, ok)
	assert.Equal(t, familyDualShock4, family)
	assert.Equal(t, 1, nth, "the second DualShock 4 connected")

	family, nth, ok = rumbleTarget(guids, 4)
	assert.True(t, ok)
	assert.Equal(t, familyDualSense, family)
	assert.Equal(t, 0, nth)

	_, _, ok = rumbleTarget(guids, 2)
	assert.False(t, ok, "an empty slot has no motors")
	_, _, ok = rumbleTarget(guids, 99)
	assert.False(t, ok)
}

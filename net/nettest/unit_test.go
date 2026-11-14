package nettest_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/net/nettest"
	"github.com/stretchr/testify/require"
)

func TestAUnitRefusesAStateOfTheWrongType(t *testing.T) {
	u := nettest.NewUnit(geom.Vec3{}, 1)
	require.PanicsWithValue(t, "nettest: Unit's net state is a Vitals, not a string", func() { u.ApplyNetState("hp") })
	require.PanicsWithValue(t, "nettest: Unit's owner state is a Purse, not a int", func() { u.ApplyOwnerState(3) })
}

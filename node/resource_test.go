package node //nolint:testpackage // white-box: shares blenderCamp with scene_test.go

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Palette struct {
	Primary string `json:"primary"`
}

func (*Palette) ResourceType() string { return "TestPalette" }

func TestMain(m *testing.M) {
	RegisterResource("TestPalette", func() Resource { return &Palette{} })
	os.Exit(m.Run())
}

type Painted struct {
	*world.BasicEntity
	Node
	Palette *Palette `node:"prop"`
}

func TestResourcePropertiesGoByIDInScenesAndInlineElsewhere(t *testing.T) {
	shared := &Palette{Primary: "red"}
	a := &Painted{BasicEntity: world.NewBasic(), Palette: shared}
	b := &Painted{BasicEntity: world.NewBasic(), Palette: shared}
	res := &Resources{}
	ra, err := GetPropertyWith(a, "palette", res)
	require.NoError(t, err)
	rb, err := GetPropertyWith(b, "palette", res)
	require.NoError(t, err)
	assert.JSONEq(t, string(ra), string(rb), "one resource, one ID")
	assert.Len(t, res.ByID, 1)

	inline, err := GetProperty(a, "palette")
	require.NoError(t, err)
	assert.JSONEq(t, `{"primary":"red"}`, string(inline))

	data := res.Data()
	back, err := LoadResources(data)
	require.NoError(t, err)
	c := &Painted{BasicEntity: world.NewBasic()}
	require.NoError(t, SetPropertyWith(c, "palette", ra, back))
	assert.Equal(t, "red", c.Palette.Primary)
	d := &Painted{BasicEntity: world.NewBasic()}
	require.NoError(t, SetPropertyWith(d, "palette", rb, back))
	assert.Same(t, c.Palette, d.Palette, "loaded references share the resource")

	require.NoError(t, SetProperty(d, "palette", json.RawMessage(`{"primary":"blue"}`)))
	assert.Equal(t, "blue", d.Palette.Primary)
	assert.NotSame(t, c.Palette, d.Palette)
	require.NoError(t, SetProperty(d, "palette", json.RawMessage(`null`)))
	assert.Nil(t, d.Palette)

	assert.ErrorContains(
		t,
		SetPropertyWith(d, "palette", json.RawMessage(`{"resource":"00000000000000aa"}`), back),
		"00000000000000aa",
	)
	_, err = LoadResources(map[string]ResourceData{"00000000000000ab": {Type: "Nope"}})
	assert.ErrorContains(t, err, "Nope")
}

func TestScenesCarryTheirResources(t *testing.T) {
	s, err := Decode([]byte(blenderCamp))
	require.NoError(t, err)
	s.Resources = map[string]ResourceData{
		"00000000000000ac": {Type: "TestPalette", Data: json.RawMessage(`{"primary":"green"}`)},
	}
	out, err := Encode(s)
	require.NoError(t, err)
	back, err := Decode(out)
	require.NoError(t, err)
	require.Contains(t, back.Resources, "00000000000000ac")
	assert.Equal(t, "TestPalette", back.Resources["00000000000000ac"].Type)
	assert.JSONEq(t, `{"primary":"green"}`, string(back.Resources["00000000000000ac"].Data))
}

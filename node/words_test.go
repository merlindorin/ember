package node_test

import (
	"encoding/json"
	"testing"

	"github.com/merlindorin/ember/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnumsAreStoredAsWords(t *testing.T) {
	b, err := json.Marshal(struct {
		Mode node.ProcessMode `json:"Mode"`
	}{node.WhenPaused})
	require.NoError(t, err)
	assert.JSONEq(t, `{"Mode":"when_paused"}`, string(b))

	var back struct {
		Mode node.ProcessMode `json:"Mode"`
	}
	require.NoError(t, json.Unmarshal(b, &back))
	assert.Equal(t, node.WhenPaused, back.Mode)
	assert.ErrorContains(
		t,
		json.Unmarshal([]byte(`{"Mode":"sleepy"}`), &back),
		"inherit, pausable, when_paused, always, disabled",
	)
	assert.Error(t, json.Unmarshal([]byte(`{"Mode":3}`), &back), "numbers are no longer read")
}

func TestFlagsAreStoredAsWordLists(t *testing.T) {
	for flags, text := range map[node.Flags]string{
		0: "none", node.Deferred: "deferred", node.Persist: "persist", node.OneShot: "one_shot",
		node.Deferred | node.OneShot: "deferred|one_shot",
	} {
		b, err := flags.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, text, string(b))
		var back node.Flags
		require.NoError(t, back.UnmarshalText(b))
		assert.Equal(t, flags, back)
	}
	var f node.Flags
	assert.Error(t, f.UnmarshalText([]byte("deferred|later")))
}

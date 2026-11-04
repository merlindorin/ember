//go:build chrome

package chrome

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shots(pictures ...string) (func() ([]byte, error), *int) {
	taken := 0
	return func() ([]byte, error) {
		p := pictures[min(taken, len(pictures)-1)]
		taken++
		return []byte(p), nil
	}, &taken
}

func TestAShotCountsOnceTwoInARowAgree(t *testing.T) {
	shoot, taken := shots("half", "whole", "whole")
	var retakes int
	got, err := steady(shoot, func(int) { retakes++ })
	require.NoError(t, err)
	assert.Equal(t, "whole", string(got))
	assert.Equal(t, 3, *taken)
	assert.Equal(t, 1, retakes, "the half-drawn shot was retaken")
}

func TestASteadyPageIsShotTwice(t *testing.T) {
	shoot, taken := shots("whole")
	got, err := steady(shoot, func(int) {})
	require.NoError(t, err)
	assert.Equal(t, "whole", string(got))
	assert.Equal(t, 2, *taken)
}

func TestShotsThatNeverAgreeGiveTheLastAfterTheTries(t *testing.T) {
	shoot, taken := shots("a", "b", "c", "d", "e")
	got, err := steady(shoot, func(int) {})
	require.NoError(t, err)
	assert.Equal(t, "d", string(got))
	assert.Equal(t, 4, *taken)
}

func TestAFailedShotFails(t *testing.T) {
	_, err := steady(func() ([]byte, error) { return nil, errors.New("no chrome") }, func(int) {})
	assert.Error(t, err)
}

func TestThePagesRunAtTwoPixelsAPointUnlessToldOtherwise(t *testing.T) {
	env := pageEnv([]string{"HOME=/h"})
	assert.Contains(t, env, "EMBER_HIDDEN=1")
	assert.Contains(t, env, "EMBER_SCALE=2", "the thresholds are 2x's, whatever the display")
	env = pageEnv([]string{"EMBER_SCALE=1"})
	assert.Contains(t, env, "EMBER_SCALE=1")
	assert.NotContains(t, env, "EMBER_SCALE=2", "a scale set is kept")
}

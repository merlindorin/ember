//go:build chrome

package chrome

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheChromeLockSaysWhoHoldsItToThoseWaiting(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	unlock, err := lockChrome()
	require.NoError(t, err)
	held, err := os.ReadFile(lockPath())
	require.NoError(t, err)
	assert.Contains(t, string(held), "pid ")

	var mu sync.Mutex
	var notes []string
	old, oldAfter := waitNote, waitAfter
	waitNote = func(s string) { mu.Lock(); notes = append(notes, s); mu.Unlock() }
	waitAfter = 10 * time.Millisecond
	defer func() { waitNote, waitAfter = old, oldAfter }()

	got := make(chan func(), 1)
	go func() {
		u, lockErr := lockChrome()
		assert.NoError(t, lockErr)
		got <- u
	}()
	time.Sleep(100 * time.Millisecond)
	select {
	case <-got:
		t.Fatal("a second run took the lock while it was held")
	default:
	}
	unlock()
	second := <-got
	second()

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, notes, "the waiting run said so")
	assert.True(
		t,
		strings.Contains(notes[0], "waiting for the Chrome lock") && strings.Contains(notes[0], string(held)),
		notes[0],
	)
	after, _ := os.ReadFile(lockPath())
	assert.Empty(t, string(after), "an unlocked file names no holder")
}

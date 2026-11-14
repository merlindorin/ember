package kit_test

import (
	"testing"

	"github.com/lantervale/ember/input"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

// liveFrames runs n frames of a Document focused on field's input and
// reports how many replayed with their live parts on top.
func liveFrames(t *testing.T, field *ui.Node, caret input.Key, n int) int {
	t.Helper()
	s := show(t, field)
	s.doc.Focus(kit.SearchInput(field))
	s.key(caret)
	s.frames(3)
	count := func() int {
		for _, c := range s.w.Stats().Counted() {
			if c.Name == "ui replays + live" {
				return c.N
			}
		}
		return 0
	}
	before := count()
	s.frames(n)
	return count() - before
}

func TestASearchFieldsCaretAwayFromItsClearButtonReplays(t *testing.T) {
	assert.Equal(t, 5, liveFrames(t, kit.SearchField("crypt"), input.KeyEnd, 5),
		"the × lies beside the caret, not over it")
}

func TestACaretUnderTheClearButtonRepaintsWhole(t *testing.T) {
	field := kit.SearchField("crypt")
	field.Children()[1].SetStyle("left: 22px; right: auto")
	assert.Zero(t, liveFrames(t, field, input.KeyHome, 5),
		"the × moved over the text's start, where the caret is: the field repaints whole")
}
